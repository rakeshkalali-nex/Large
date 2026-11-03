package sessions_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"buildestate/backend/internal/access/csrf"
	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/httpapi"
)

type sessionBrowser struct {
	handler http.Handler
	cookies map[string]*http.Cookie
	csrf    string
}

func newSessionBrowser(t *testing.T, f *sessionFixture) *sessionBrowser {
	t.Helper()
	return &sessionBrowser{
		handler: httpapi.New(f.deps, t.TempDir(), t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil))),
		cookies: make(map[string]*http.Cookie),
	}
}

func (b *sessionBrowser) request(method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://app.example.test"+path, strings.NewReader(body))
	r.RemoteAddr = "192.0.2.10:12345"
	r.Header.Set("Content-Type", "application/json")
	if method != http.MethodGet {
		r.Header.Set("Origin", "http://app.example.test")
		if b.csrf != "" {
			r.Header.Set(csrf.HeaderName, b.csrf)
		}
	}
	for _, cookie := range b.cookies {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	b.handler.ServeHTTP(w, r)
	for _, cookie := range w.Result().Cookies() {
		if cookie.MaxAge < 0 {
			delete(b.cookies, cookie.Name)
		} else {
			b.cookies[cookie.Name] = cookie
		}
	}
	return w
}

func requireHTTPStatus(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status=%d, want %d; body=%s", w.Code, status, w.Body.String())
	}
}

func decodeSessionData(t *testing.T, w *httptest.ResponseRecorder, dst any) {
	t.Helper()
	var envelope struct {
		Data      json.RawMessage `json:"data"`
		RequestID string          `json:"requestId"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.RequestID == "" || len(envelope.Data) == 0 {
		t.Fatal("missing API envelope fields")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("authentication response must not be cached")
	}
	if err := json.Unmarshal(envelope.Data, dst); err != nil {
		t.Fatal(err)
	}
}

func requireHTTPError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	requireHTTPStatus(t, w, status)
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.Code != code || envelope.Error.Message == "" || envelope.RequestID == "" {
		t.Fatalf("incorrect safe error envelope: %s", w.Body.String())
	}
}

func (b *sessionBrowser) bootstrap(t *testing.T) {
	t.Helper()
	w := b.request(http.MethodGet, "/api/v1/auth/csrf", "")
	requireHTTPStatus(t, w, 200)
	var data struct {
		Token     string `json:"csrfToken"`
		ExpiresAt string `json:"expiresAt"`
	}
	decodeSessionData(t, w, &data)
	if data.Token == "" {
		t.Fatal("missing CSRF nonce")
	}
	if _, err := time.Parse(time.RFC3339Nano, data.ExpiresAt); err != nil {
		t.Fatalf("invalid CSRF expiration: %v", err)
	}
	cookie := b.cookies[csrf.CookieName]
	if cookie == nil || cookie.Value != data.Token || cookie.HttpOnly || cookie.Domain != "" || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("incorrect readable host-only CSRF cookie")
	}
	b.csrf = data.Token
}

func (b *sessionBrowser) login(t *testing.T, f *sessionFixture) sessions.Identity {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": "staff@example.test", "password": f.password})
	if err != nil {
		t.Fatal(err)
	}
	w := b.request(http.MethodPost, "/api/v1/auth/login", string(body))
	requireHTTPStatus(t, w, 200)
	var identity sessions.Identity
	decodeSessionData(t, w, &identity)
	if identity.CSRFToken == "" || identity.CSRFToken == b.csrf {
		t.Fatal("login did not rotate CSRF token")
	}
	b.csrf = identity.CSRFToken
	if strings.Contains(w.Body.String(), f.password) || strings.Contains(w.Body.String(), "$argon2id$") {
		t.Fatal("identity response leaked credentials")
	}
	return identity
}

func TestHTTPUnauthenticatedErrorEnvelope(t *testing.T) {
	f := newSessionFixture(t, "site-engineer")
	for _, path := range []string{"/api/v1/auth/me", "/api/v1/projects"} {
		t.Run(path, func(t *testing.T) {
			b := newSessionBrowser(t, f)
			w := b.request(http.MethodGet, path, "")
			requireHTTPError(t, w, http.StatusUnauthorized, "AUTH_REQUIRED")
			if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
				t.Fatalf("expected JSON error, got Content-Type %q", got)
			}
		})
	}
}

func TestHTTPNormalLoginMeRotationAndLogoutContract(t *testing.T) {
	f := newSessionFixture(t, "site-engineer")
	b := newSessionBrowser(t, f)
	b.bootstrap(t)
	identity := b.login(t, f)
	if identity.User.ID != "staff" || identity.User.Role != "site-engineer" || identity.LandingPath != "/site-logs" || identity.SetupRequired || identity.ProjectIDs == nil || identity.Capabilities == nil {
		t.Fatalf("incorrect login identity: %+v", identity)
	}
	cookie := b.cookies[sessions.CookieName]
	if cookie == nil || !cookie.HttpOnly || cookie.Secure || cookie.Domain != "" || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("incorrect local HTTP session cookie policy")
	}
	first := cookie.Value
	last, idle, absolute := f.deadlines(t, first)
	f.clock.now = f.clock.now.Add(10 * time.Minute)
	w := b.request(http.MethodGet, "/api/v1/auth/me?role=administrator&projectId=forged", "")
	requireHTTPStatus(t, w, 200)
	var restored sessions.Identity
	decodeSessionData(t, w, &restored)
	if restored.User.Role != "site-engineer" || restored.User.ID != identity.User.ID || restored.IdleExpiresAt != idle || restored.AbsoluteExpiresAt != absolute {
		t.Fatal("me trusted forged claims or altered expiry")
	}
	b.csrf = restored.CSRFToken
	a, c, d := f.deadlines(t, first)
	if a != last || c != idle || d != absolute {
		t.Fatal("HTTP me refreshed idle lifetime")
	}
	b.login(t, f)
	second := b.cookies[sessions.CookieName].Value
	if second == first {
		t.Fatal("HTTP login failed to rotate session")
	}
	_, err := f.service.Restore(context.Background(), first)
	requireSessionError(t, err, 401, "SESSION_EXPIRED")
	w = b.request(http.MethodPost, "/api/v1/auth/logout", `{}`)
	requireHTTPStatus(t, w, http.StatusNoContent)
	if w.Body.Len() != 0 {
		t.Fatal("logout 204 must have no response body")
	}
	if b.cookies[sessions.CookieName] != nil || b.cookies[csrf.CookieName] != nil {
		t.Fatal("logout did not clear both cookies")
	}
	_, err = f.service.Restore(context.Background(), second)
	requireSessionError(t, err, 401, "SESSION_EXPIRED")
	w = b.request(http.MethodGet, "/api/v1/auth/me", "")
	requireHTTPError(t, w, http.StatusUnauthorized, "AUTH_REQUIRED")
}

func TestHTTPLoginRejectsUnknownFieldsAndForgedRoles(t *testing.T) {
	f := newSessionFixture(t, "site-engineer")
	b := newSessionBrowser(t, f)
	b.bootstrap(t)
	for _, extra := range []map[string]any{{"role": "administrator"}, {"projectIds": []string{"private-project"}}, {"unexpected": true}} {
		extra["email"], extra["password"] = "staff@example.test", f.password
		body, err := json.Marshal(extra)
		if err != nil {
			t.Fatal(err)
		}
		requireHTTPError(t, b.request(http.MethodPost, "/api/v1/auth/login", string(body)), 400, "MALFORMED_REQUEST")
	}
	var count int
	if err := f.db.QueryRow(`SELECT count(*) FROM sessions WHERE user_id='staff'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("forged login input created a session")
	}
	if b.login(t, f).User.Role != "site-engineer" {
		t.Fatal("forged role changed persisted identity")
	}
}

func TestHTTPLoginAndLogoutRequireValidSessionBoundCSRF(t *testing.T) {
	f := newSessionFixture(t, "administrator")
	b := newSessionBrowser(t, f)
	body, err := json.Marshal(map[string]string{"email": "staff@example.test", "password": f.password})
	if err != nil {
		t.Fatal(err)
	}
	requireHTTPError(t, b.request(http.MethodPost, "/api/v1/auth/login", string(body)), 403, "CSRF_INVALID")
	b.bootstrap(t)
	oldNonce, oldCookie := b.csrf, b.cookies[csrf.CookieName]
	r := httptest.NewRequest(http.MethodPost, "http://app.example.test/api/v1/auth/login", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://attacker.example")
	r.Header.Set(csrf.HeaderName, b.csrf)
	r.AddCookie(oldCookie)
	w := httptest.NewRecorder()
	b.handler.ServeHTTP(w, r)
	requireHTTPError(t, w, 403, "CSRF_INVALID")
	b.login(t, f)
	token := b.cookies[sessions.CookieName].Value
	boundNonce, boundCookie := b.csrf, b.cookies[csrf.CookieName]
	b.csrf, b.cookies[csrf.CookieName] = oldNonce, oldCookie
	requireHTTPError(t, b.request(http.MethodPost, "/api/v1/auth/logout", `{}`), 403, "CSRF_INVALID")
	if _, err := f.service.Restore(context.Background(), token); err != nil {
		t.Fatal("invalid CSRF revoked valid session")
	}
	b.csrf, b.cookies[csrf.CookieName] = boundNonce, boundCookie
	requireHTTPStatus(t, b.request(http.MethodPost, "/api/v1/auth/logout", `{}`), 204)
}

func TestHTTPExpiredMeClearsCookieAndLogoutRemainsAvailable(t *testing.T) {
	f := newSessionFixture(t, "administrator")
	b := newSessionBrowser(t, f)
	b.bootstrap(t)
	b.login(t, f)
	token := b.cookies[sessions.CookieName].Value
	f.clock.now = f.clock.now.Add(30 * time.Minute)
	w := b.request(http.MethodGet, "/api/v1/auth/me", "")
	requireHTTPError(t, w, 401, "SESSION_EXPIRED")
	if b.cookies[sessions.CookieName] != nil {
		t.Fatal("expired me did not clear cookie")
	}
	// A stale browser can obtain fresh CSRF protection and explicitly log out.
	b.cookies[sessions.CookieName] = &http.Cookie{Name: sessions.CookieName, Value: token}
	b.bootstrap(t)
	requireHTTPStatus(t, b.request(http.MethodPost, "/api/v1/auth/logout", `{}`), 204)
}

func TestHTTPSetupRequiredIdentityAndSecureCookie(t *testing.T) {
	f := newSessionFixture(t, "administrator")
	if _, err := f.db.Exec(`UPDATE users SET status='setup-required',setup_required=1 WHERE id='staff'`); err != nil {
		t.Fatal(err)
	}
	b := newSessionBrowser(t, f)
	b.bootstrap(t)
	identity := b.login(t, f)
	if !identity.SetupRequired || len(identity.Capabilities) != 0 || len(identity.ProjectIDs) != 0 {
		t.Fatal("setup identity leaked domain grants")
	}
	w := b.request(http.MethodGet, "/api/v1/auth/me", "")
	requireHTTPStatus(t, w, 200)
	decodeSessionData(t, w, &identity)
	b.csrf = identity.CSRFToken
	requireHTTPStatus(t, b.request(http.MethodPost, "/api/v1/auth/logout", `{}`), 204)

	// Exercise HTTPS directly through the real composition root, not a test route.
	r := httptest.NewRequest(http.MethodGet, "https://app.example.test/api/v1/auth/csrf", nil)
	w = httptest.NewRecorder()
	b.handler.ServeHTTP(w, r)
	requireHTTPStatus(t, w, 200)
	var data struct {
		Token string `json:"csrfToken"`
	}
	decodeSessionData(t, w, &data)
	body, err := json.Marshal(map[string]string{"email": "staff@example.test", "password": f.password})
	if err != nil {
		t.Fatal(err)
	}
	r = httptest.NewRequest(http.MethodPost, "https://app.example.test/api/v1/auth/login", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://app.example.test")
	r.Header.Set(csrf.HeaderName, data.Token)
	for _, cookie := range w.Result().Cookies() {
		r.AddCookie(cookie)
	}
	w = httptest.NewRecorder()
	b.handler.ServeHTTP(w, r)
	requireHTTPStatus(t, w, 200)
	found := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == sessions.CookieName {
			found = true
			if !cookie.Secure || !cookie.HttpOnly || cookie.Domain != "" || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode {
				t.Fatal("HTTPS session cookie is not protected")
			}
		}
	}
	if !found {
		t.Fatal("HTTPS login did not issue a session cookie")
	}
}
