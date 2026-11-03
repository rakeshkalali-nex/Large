package recovery_test

import (
	"buildestate/backend/internal/access"
	"buildestate/backend/internal/access/passwords"
	"buildestate/backend/internal/access/recovery"
	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/testsupport"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

type fixture struct {
	db      *sql.DB
	d       contracts.Dependencies
	c       *clock
	s       *recovery.Service
	session string
}

func fixtureFor(t *testing.T) *fixture {
	db := testsupport.Database(t)
	c := &clock{time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)}
	password := transactions.NewID()
	hash, e := sessions.HashPassword(password)
	if e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec(`INSERT INTO users(id,display_name,email,normalized_email,role_code,status,setup_required,password_hash) VALUES('admin','Admin','admin@test.invalid','admin@test.invalid','administrator','active',0,?); INSERT INTO users(id,display_name,email,normalized_email,role_code) VALUES('target','Target','target@test.invalid','target@test.invalid','site-engineer')`, hash)
	if e != nil {
		t.Fatal(e)
	}
	d := contracts.Dependencies{DB: db, Clock: c, SessionKey: []byte(strings.Repeat("k", 32))}
	s := recovery.New(d)
	token, _, e := s.Sessions.Login(context.Background(), "admin@test.invalid", password, "")
	if e != nil {
		t.Fatal(e)
	}
	return &fixture{db, d, c, s, token}
}
func input(purpose string) recovery.Input {
	return recovery.Input{Version: 1, Purpose: purpose, IdentityVerified: true, Reason: "Identity verified in person"}
}
func want(t *testing.T, e error, c string) {
	t.Helper()
	var v *transactions.Error
	if !errors.As(e, &v) || v.Code != c {
		t.Fatalf("want %s got %v", c, e)
	}
}
func TestSetupReplacementExpiryAndCompletion(t *testing.T) {
	f := fixtureFor(t)
	ctx := context.Background()
	first, _, e := f.s.Issue(ctx, f.session, "target", transactions.NewID(), input("setup"))
	if e != nil {
		t.Fatal(e)
	}
	expiry, _ := time.Parse(time.RFC3339Nano, first.ExpiresAt)
	if expiry.Sub(f.c.now) != 24*time.Hour {
		t.Fatal("wrong setup lifetime")
	}
	second, _, e := f.s.Issue(ctx, f.session, "target", transactions.NewID(), input("setup"))
	if e != nil {
		t.Fatal(e)
	}
	p := passwords.New(f.d)
	want(t, p.Reset(ctx, strings.TrimPrefix(first.ResetLink, "/login#token="), transactions.NewID(), transactions.NewID()), "INVALID_RESET_TOKEN")
	token := strings.TrimPrefix(second.ResetLink, "/login#token=")
	var digest string
	f.db.QueryRow(`SELECT token_digest FROM password_tokens WHERE revoked_at IS NULL`).Scan(&digest)
	if digest != sessions.Digest(token) || digest == token {
		t.Fatal("not hash only")
	}
	if e = p.Reset(ctx, token, transactions.NewID(), transactions.NewID()); e != nil {
		t.Fatal(e)
	}
	var n int
	f.db.QueryRow(`SELECT count(*) FROM users WHERE id='target' AND status='active' AND setup_required=0`).Scan(&n)
	if n != 1 {
		t.Fatal("setup not completed")
	}
	var action string
	f.db.QueryRow(`SELECT action FROM audit_events WHERE action='auth.setup.completed'`).Scan(&action)
	if action == "" {
		t.Fatal("missing setup audit")
	}
}
func TestRecoveryTTLAndIssuanceControls(t *testing.T) {
	f := fixtureFor(t)
	ctx := context.Background()
	key := transactions.NewID()
	out, _, e := f.s.Issue(ctx, f.session, "target", key, input("recovery"))
	if e != nil {
		t.Fatal(e)
	}
	expiry, _ := time.Parse(time.RFC3339Nano, out.ExpiresAt)
	if expiry.Sub(f.c.now) != 30*time.Minute {
		t.Fatal("wrong recovery lifetime")
	}
	_, _, e = f.s.Issue(ctx, f.session, "target", key, input("recovery"))
	want(t, e, "IDEMPOTENCY_MISMATCH")
	in := input("recovery")
	in.IdentityVerified = false
	_, _, e = f.s.Issue(ctx, f.session, "target", transactions.NewID(), in)
	want(t, e, "VALIDATION_FAILED")
	in = input("recovery")
	in.Version = 2
	_, _, e = f.s.Issue(ctx, f.session, "target", transactions.NewID(), in)
	want(t, e, "STALE_VERSION")
	f.db.Exec(`UPDATE users SET status='active',setup_required=0 WHERE id='target'`)
	_, _, e = f.s.Issue(ctx, f.session, "target", transactions.NewID(), input("setup"))
	want(t, e, "INVALID_TRANSITION")
	f.db.Exec(`UPDATE users SET role_code='site-engineer' WHERE id='admin'`)
	_, _, e = f.s.Issue(ctx, f.session, "target", transactions.NewID(), input("recovery"))
	want(t, e, "FORBIDDEN")
}
func TestIssuanceRateLimitAndAuditRollback(t *testing.T) {
	f := fixtureFor(t)
	ctx := context.Background()
	// Unsafe reason fails the existing audit writer and rolls back token/idempotency.
	in := input("recovery")
	in.Reason = "password=do-not-store-this"
	if _, _, e := f.s.Issue(ctx, f.session, "target", transactions.NewID(), in); e == nil {
		t.Fatal("unsafe reason accepted")
	}
	var n int
	f.db.QueryRow(`SELECT count(*) FROM password_tokens`).Scan(&n)
	if n != 0 {
		t.Fatal("failed audit left token")
	}
	f.db.QueryRow(`SELECT count(*) FROM idempotency_records`).Scan(&n)
	if n != 0 {
		t.Fatal("failed audit left replay record")
	}
	for i := 0; i < 4; i++ {
		if _, _, e := f.s.Issue(ctx, f.session, "target", transactions.NewID(), input("recovery")); e != nil {
			t.Fatal(e)
		}
	}
	_, wait, e := f.s.Issue(ctx, f.session, "target", transactions.NewID(), input("recovery"))
	want(t, e, "RATE_LIMITED")
	if wait != time.Hour {
		t.Fatalf("retry %v", wait)
	}
}
func TestHTTPGuidanceAndAdministratorIssuance(t *testing.T) {
	f := fixtureFor(t)
	mux := http.NewServeMux()
	h := access.RegisterRoutes(mux, f.d)
	handler := h.Protect(mux)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "http://app.test/api/v1/auth/recovery-guidance", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "identity verification") {
		t.Fatalf("guidance %d %s", w.Code, w.Body.String())
	}
	r := httptest.NewRequest("GET", "http://app.test/api/v1/auth/csrf", nil)
	r.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: f.session})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	var env struct {
		Data struct {
			Token string `json:"csrfToken"`
		}
	}
	json.Unmarshal(w.Body.Bytes(), &env)
	cookies := w.Result().Cookies()
	body, _ := json.Marshal(input("setup"))
	r = httptest.NewRequest("POST", "http://app.test/api/v1/users/target/password-reset", strings.NewReader(string(body)))
	r.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: f.session})
	for _, c := range cookies {
		r.AddCookie(c)
	}
	r.Header.Set("Origin", "http://app.test")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", env.Data.Token)
	r.Header.Set("Idempotency-Key", transactions.NewID())
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "/login#token=") {
		t.Fatalf("issuance %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("link cached")
	}
}
