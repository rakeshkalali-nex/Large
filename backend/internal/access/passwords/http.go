package passwords

import (
	"buildestate/backend/internal/access/csrf"
	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
	"net/http"
)

func SessionToken(r *http.Request) string {
	c, e := r.Cookie(sessions.CookieName)
	if e != nil {
		return ""
	}
	return c.Value
}

// BrowserMutation is also applied by the common gate; keeping CSRF here makes
// these endpoints safe when mounted independently in a service harness.
func BrowserMutation(w http.ResponseWriter, r *http.Request, d contracts.Dependencies) bool {
	if d.DB == nil {
		transport.Unavailable(w, r)
		return false
	}
	binding := ""
	if t := SessionToken(r); t != "" {
		binding = sessions.Digest(t)
	}
	c := csrf.Manager{Key: d.SessionKey, Clock: sessions.New(d).Clock, Secure: d.CookieSecure}
	if err := c.Validate(r, binding); err != nil {
		transport.Error(w, r, 403, "CSRF_INVALID", "Refresh the page and try again.")
		return false
	}
	if !ValidKey(r.Header.Get("Idempotency-Key")) {
		transport.Error(w, r, 422, "VALIDATION_FAILED", "A UUID Idempotency-Key is required.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	w.Header().Set("Referrer-Policy", "no-referrer")
	return true
}
func RegisterRoutes(mux *http.ServeMux, d contracts.Dependencies) {
	s := New(d)
	finish := func(w http.ResponseWriter, r *http.Request, err error) {
		if err != nil {
			transactions.WriteError(w, r, err)
			return
		}
		for _, name := range []string{sessions.CookieName, "buildestate_csrf"} {
			http.SetCookie(w, &http.Cookie{Name: name, Path: "/", MaxAge: -1, HttpOnly: name == sessions.CookieName, SameSite: http.SameSiteLaxMode, Secure: d.CookieSecure || r.TLS != nil})
		}
		transport.JSON(w, r, 200, map[string]bool{"passwordChanged": true, "signInRequired": true})
	}
	mux.HandleFunc("POST /api/v1/auth/reset-password", func(w http.ResponseWriter, r *http.Request) {
		if !BrowserMutation(w, r, d) {
			return
		}
		var in struct {
			Token       string `json:"token"`
			NewPassword string `json:"newPassword"`
		}
		if !transport.DecodeJSON(w, r, &in) {
			return
		}
		finish(w, r, s.Reset(r.Context(), in.Token, in.NewPassword, r.Header.Get("Idempotency-Key")))
	})
	mux.HandleFunc("POST /api/v1/auth/change-password", func(w http.ResponseWriter, r *http.Request) {
		if !BrowserMutation(w, r, d) {
			return
		}
		var in struct {
			CurrentPassword string `json:"currentPassword"`
			NewPassword     string `json:"newPassword"`
		}
		if !transport.DecodeJSON(w, r, &in) {
			return
		}
		finish(w, r, s.Change(r.Context(), SessionToken(r), in.CurrentPassword, in.NewPassword, r.Header.Get("Idempotency-Key")))
	})
}
