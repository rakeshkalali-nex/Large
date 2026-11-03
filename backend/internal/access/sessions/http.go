package sessions

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"buildestate/backend/internal/access/csrf"
	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/access/ratelimit"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
)

type Handler struct {
	Service *Service
	CSRF    csrf.Manager
	Limits  ratelimit.Limiter
	Secure  bool
}

func NewHandler(d contracts.Dependencies) *Handler {
	s := New(d)
	return &Handler{Service: s, CSRF: csrf.Manager{Key: d.SessionKey, Clock: s.Clock, Secure: d.CookieSecure}, Limits: ratelimit.Limiter{DB: d.DB, Key: d.SessionKey, Clock: s.Clock}, Secure: d.CookieSecure}
}
func tokenFrom(r *http.Request) string {
	c, e := r.Cookie(CookieName)
	if e != nil {
		return ""
	}
	return c.Value
}
func binding(r *http.Request) string {
	token := tokenFrom(r)
	if token == "" {
		return ""
	}
	return Digest(token)
}
func (h *Handler) cookie(w http.ResponseWriter, r *http.Request, token string) {
	c := &http.Cookie{Name: CookieName, Value: token, Path: "/", HttpOnly: true, Secure: h.Secure || r.TLS != nil, SameSite: http.SameSiteLaxMode}
	if token == "" {
		c.MaxAge = -1
		c.Expires = time.Unix(1, 0)
	}
	http.SetCookie(w, c)
}
func (h *Handler) failure(w http.ResponseWriter, r *http.Request, err error) {
	var safe *transactions.Error
	if errors.As(err, &safe) && safe.Status == 401 {
		h.cookie(w, r, "")
	}
	transactions.WriteError(w, r, err)
}
func (h *Handler) csrfOK(w http.ResponseWriter, r *http.Request) bool {
	if err := h.CSRF.Validate(r, binding(r)); err != nil {
		transport.Error(w, r, 403, "CSRF_INVALID", "Refresh the page and try again.")
		return false
	}
	return true
}
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/csrf", func(w http.ResponseWriter, r *http.Request) {
		token, err := h.CSRF.Issue(w, r, binding(r))
		if err != nil {
			transport.Unavailable(w, r)
			return
		}
		transport.JSON(w, r, 200, map[string]string{"csrfToken": token, "expiresAt": h.Service.Clock.Now().UTC().Add(csrf.Lifetime).Format(timestamp)})
	})
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("GET /api/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
		if h.Service.DB == nil {
			transport.Unavailable(w, r)
			return
		}
		identity, err := h.Service.Restore(r.Context(), tokenFrom(r))
		if err != nil {
			h.failure(w, r, err)
			return
		}
		identity.CSRFToken, err = h.CSRF.Issue(w, r, binding(r))
		if err != nil {
			transport.Unavailable(w, r)
			return
		}
		transport.JSON(w, r, 200, identity)
	})
	mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if h.Service.DB == nil {
			transport.Unavailable(w, r)
			return
		}
		if !h.csrfOK(w, r) {
			return
		}
		if err := h.Service.Logout(r.Context(), tokenFrom(r)); err != nil {
			h.failure(w, r, err)
			return
		}
		h.cookie(w, r, "")
		http.SetCookie(w, &http.Cookie{Name: "buildestate_csrf", Path: "/", MaxAge: -1, Secure: h.Secure || r.TLS != nil, SameSite: http.SameSiteLaxMode})
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNoContent)
	})
}
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if h.Service.DB == nil {
		transport.Unavailable(w, r)
		return
	}
	if !h.csrfOK(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !transport.DecodeJSON(w, r, &input) {
		return
	}
	if len(input.Email) > 320 || !utf8.ValidString(input.Password) || utf8.RuneCountInString(input.Password) > 128 {
		transport.Error(w, r, 400, "MALFORMED_REQUEST", "Request body is invalid.")
		return
	}
	origin, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		origin = r.RemoteAddr
	}
	wait, err := h.Limits.Attempt(r.Context(), "login", input.Email, origin)
	if err != nil {
		transport.Unavailable(w, r)
		return
	}
	if wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		transport.Error(w, r, 429, "RATE_LIMITED", "Too many attempts. Try again later.")
		return
	}
	token, identity, err := h.Service.Login(r.Context(), input.Email, input.Password, tokenFrom(r))
	if err != nil {
		h.failure(w, r, err)
		return
	}
	identity.CSRFToken, err = h.CSRF.Issue(w, r, Digest(token))
	if err != nil {
		_ = h.Service.Logout(r.Context(), token)
		transport.Unavailable(w, r)
		return
	}
	h.cookie(w, r, token)
	transport.JSON(w, r, 200, identity)
}

// Protect is the common API gate. Resource handlers must additionally use the
// scoped policy/transaction helpers before querying records or committing writes.
func (h *Handler) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") || strings.HasPrefix(r.URL.Path, "/api/v1/health/") {
			next.ServeHTTP(w, r)
			return
		}
		public := r.URL.Path == "/api/v1/auth/csrf" || r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/me" || r.URL.Path == "/api/v1/auth/logout" || r.URL.Path == "/api/v1/auth/recovery-guidance"
		if public {
			next.ServeHTTP(w, r)
			return
		}
		// Reset completion has its own token authentication; still requires CSRF.
		if r.URL.Path == "/api/v1/auth/reset-password" {
			if !h.csrfOK(w, r) {
				return
			}
			if !h.limitReset(w, r) {
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if h.Service.DB == nil {
			transport.Unavailable(w, r)
			return
		}
		identity, err := h.Service.Restore(r.Context(), tokenFrom(r))
		if err != nil {
			h.failure(w, r, err)
			return
		}
		if identity.SetupRequired {
			transport.Error(w, r, 403, "SETUP_REQUIRED", "Complete password setup before continuing.")
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			if !h.csrfOK(w, r) {
				return
			}
		}
		ctx := policy.WithPrincipal(r.Context(), policy.Principal{UserID: identity.User.ID, SessionDigest: Digest(tokenFrom(r))})
		capability, individual, known := policy.RouteCapability(r.Method, r.URL.Path)
		if !known {
			transport.Error(w, r, 403, "FORBIDDEN", "Access is denied.")
			return
		}
		err = (policy.Service{DB: h.Service.DB, Clock: h.Service.Clock}).Authorize(ctx, capability, "")
		if err != nil {
			var safe *transactions.Error
			if individual && errors.As(err, &safe) && safe.Status == http.StatusForbidden {
				transactions.WriteError(w, r, policy.NotFound())
			} else {
				h.failure(w, r, err)
			}
			return
		}
		response := &interactionResponse{ResponseWriter: w}
		next.ServeHTTP(response, r.WithContext(ctx))
		// Explicit UI interaction only. Dashboard/status polling must omit this
		// marker; it is a liveness hint, never an authorization claim.
		if response.status >= 200 && response.status < 300 && r.Header.Get("X-User-Interaction") == "true" {
			_ = h.Service.Touch(ctx, tokenFrom(r))
		}
	})
}

type interactionResponse struct {
	http.ResponseWriter
	status int
}

func (w *interactionResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *interactionResponse) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
func (w *interactionResponse) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// The reset route owns credential validation; this gate only extracts its token
// subject for throttling and restores the bounded body unchanged for that route.
func (h *Handler) limitReset(w http.ResponseWriter, r *http.Request) bool {
	if h.Service.DB == nil {
		transport.Unavailable(w, r)
		return false
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8192))
	if err != nil {
		transport.Error(w, r, 413, "BODY_TOO_LARGE", "Request body is too large.")
		return false
	}
	var subject struct {
		Token string `json:"token"`
	}
	if json.Unmarshal(body, &subject) != nil {
		transport.Error(w, r, 400, "MALFORMED_REQUEST", "Request body is invalid.")
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	origin, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		origin = r.RemoteAddr
	}
	wait, err := h.Limits.Attempt(r.Context(), "reset", Digest(subject.Token), origin)
	if err != nil {
		transport.Unavailable(w, r)
		return false
	}
	if wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		transport.Error(w, r, 429, "RATE_LIMITED", "Too many attempts. Try again later.")
		return false
	}
	return true
}
