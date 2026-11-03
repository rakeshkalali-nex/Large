package passwords_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"buildestate/backend/internal/access"
	"buildestate/backend/internal/access/passwords"
	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/testsupport"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }

type failingAudit struct{}

func (failingAudit) Append(context.Context, contracts.Transaction, contracts.AuditEvent) error {
	return errors.New("injected audit failure")
}

type fixture struct {
	db                       *sql.DB
	c                        *testClock
	d                        contracts.Dependencies
	s                        *passwords.Service
	password, token, session string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testsupport.Database(t)
	c := &testClock{time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)}
	password := transactions.NewID()
	hash, e := sessions.HashPassword(password)
	if e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec(`INSERT INTO users(id,display_name,email,normalized_email,role_code,status,setup_required,password_hash) VALUES('staff','Staff','staff@example.test','staff@example.test','site-engineer','active',0,?)`, hash)
	if e != nil {
		t.Fatal(e)
	}
	d := contracts.Dependencies{DB: db, Clock: c, SessionKey: []byte(strings.Repeat("k", 32))}
	s := passwords.New(d)
	session, _, e := s.Sessions.Login(context.Background(), "staff@example.test", password, "")
	if e != nil {
		t.Fatal(e)
	}
	token := strings.Repeat("a", 43)
	_, e = db.Exec(`INSERT INTO password_tokens(id,token_digest,user_id,kind,issuer_id,created_at,expires_at,identity_verified,reason) VALUES('token',?,'staff','recovery','staff',?,?,1,'Verified in person')`, sessions.Digest(token), transactions.Timestamp(c.now), transactions.Timestamp(c.now.Add(30*time.Minute)))
	if e != nil {
		t.Fatal(e)
	}
	return &fixture{db, c, d, s, password, token, session}
}
func code(t *testing.T, err error, want string) {
	t.Helper()
	var e *transactions.Error
	if !errors.As(err, &e) || e.Code != want {
		t.Fatalf("want %s got %v", want, err)
	}
}
func TestResetAtomicSingleUseAndSecretFree(t *testing.T) {
	f := newFixture(t)
	next := transactions.NewID()
	if e := f.s.Reset(context.Background(), f.token, next, transactions.NewID()); e != nil {
		t.Fatal(e)
	}
	code(t, f.s.Reset(context.Background(), f.token, next, transactions.NewID()), "INVALID_RESET_TOKEN")
	if _, e := f.s.Sessions.Restore(context.Background(), f.session); e == nil {
		t.Fatal("old session survived")
	}
	var hash string
	var setup int
	f.db.QueryRow(`SELECT password_hash,setup_required FROM users WHERE id='staff'`).Scan(&hash, &setup)
	if !sessions.VerifyPassword(hash, next) || setup != 0 {
		t.Fatal("password not completed")
	}
	var consumed sql.NullString
	f.db.QueryRow(`SELECT consumed_at FROM password_tokens WHERE id='token'`).Scan(&consumed)
	if !consumed.Valid {
		t.Fatal("token not consumed")
	}
	var audit string
	f.db.QueryRow(`SELECT action FROM audit_events WHERE action='auth.password.reset.completed'`).Scan(&audit)
	if audit == "" {
		t.Fatal("missing audit")
	}
	rows, e := f.db.Query(`SELECT request_digest || result_id FROM idempotency_records`)
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		rows.Scan(&v)
		if strings.Contains(v, next) || strings.Contains(v, f.token) {
			t.Fatal("persisted secret")
		}
	}
}
func TestInvalidTokenStatesAndPasswordPolicy(t *testing.T) {
	for _, state := range []string{"unknown", "expired", "revoked", "consumed", "inactive"} {
		t.Run(state, func(t *testing.T) {
			f := newFixture(t)
			switch state {
			case "unknown":
				f.token = strings.Repeat("b", 43)
			case "expired":
				f.c.now = f.c.now.Add(30 * time.Minute)
			case "revoked":
				f.db.Exec(`UPDATE password_tokens SET revoked_at=created_at`)
			case "consumed":
				f.db.Exec(`UPDATE password_tokens SET consumed_at=created_at`)
			case "inactive":
				f.db.Exec(`UPDATE users SET status='deactivated'`)
			}
			code(t, f.s.Reset(context.Background(), f.token, transactions.NewID(), transactions.NewID()), "INVALID_RESET_TOKEN")
			var hash string
			f.db.QueryRow(`SELECT password_hash FROM users WHERE id='staff'`).Scan(&hash)
			if !sessions.VerifyPassword(hash, f.password) {
				t.Fatal("invalid token changed password")
			}
		})
	}
	f := newFixture(t)
	for _, p := range []string{"short", strings.Repeat("a", 129)} {
		code(t, f.s.Reset(context.Background(), f.token, p, transactions.NewID()), "VALIDATION_FAILED")
	}
	if e := f.s.Reset(context.Background(), f.token, strings.Repeat("界", 12), transactions.NewID()); e != nil {
		t.Fatal(e)
	}
}
func TestConcurrentResetOneWinner(t *testing.T) {
	f := newFixture(t)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- f.s.Reset(context.Background(), f.token, transactions.NewID(), transactions.NewID())
		}()
	}
	wg.Wait()
	close(results)
	wins := 0
	for err := range results {
		if err == nil {
			wins++
		} else {
			code(t, err, "INVALID_RESET_TOKEN")
		}
	}
	if wins != 1 {
		t.Fatalf("winners %d", wins)
	}
}
func TestAuditFailureRollsBackEveryEffect(t *testing.T) {
	f := newFixture(t)
	f.s.Sessions.Audit = failingAudit{}
	if e := f.s.Reset(context.Background(), f.token, transactions.NewID(), transactions.NewID()); e == nil {
		t.Fatal("expected failure")
	}
	var n int
	f.db.QueryRow(`SELECT count(*) FROM password_tokens WHERE consumed_at IS NULL AND revoked_at IS NULL`).Scan(&n)
	if n != 1 {
		t.Fatal("token lost")
	}
	f.db.QueryRow(`SELECT count(*) FROM idempotency_records`).Scan(&n)
	if n != 0 {
		t.Fatal("idempotency survived rollback")
	}
	if _, e := f.s.Sessions.Restore(context.Background(), f.session); e != nil {
		t.Fatal(e)
	}
	var hash string
	f.db.QueryRow(`SELECT password_hash FROM users WHERE id='staff'`).Scan(&hash)
	if !sessions.VerifyPassword(hash, f.password) {
		t.Fatal("password survived rollback")
	}
}
func TestAuthenticatedChange(t *testing.T) {
	f := newFixture(t)
	next := transactions.NewID()
	code(t, f.s.Change(context.Background(), f.session, "wrong", next, transactions.NewID()), "INVALID_CREDENTIALS")
	if e := f.s.Change(context.Background(), f.session, f.password, next, transactions.NewID()); e != nil {
		t.Fatal(e)
	}
	code(t, f.s.Reset(context.Background(), f.token, next, transactions.NewID()), "INVALID_RESET_TOKEN")
	if _, e := f.s.Sessions.Restore(context.Background(), f.session); e == nil {
		t.Fatal("session survived change")
	}
	if _, _, e := f.s.Sessions.Login(context.Background(), "staff@example.test", next, ""); e != nil {
		t.Fatal(e)
	}
}
func TestResetHTTPContractCSRFAndLimiter(t *testing.T) {
	f := newFixture(t)
	mux := http.NewServeMux()
	h := access.RegisterRoutes(mux, f.d)
	handler := h.Protect(mux)
	bootstrap := httptest.NewRecorder()
	handler.ServeHTTP(bootstrap, httptest.NewRequest("GET", "http://app.test/api/v1/auth/csrf", nil))
	var envelope struct {
		Data struct {
			Token string `json:"csrfToken"`
		}
	}
	if e := json.Unmarshal(bootstrap.Body.Bytes(), &envelope); e != nil {
		t.Fatal(e)
	}
	call := func(token string, csrf bool, key string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"token": token, "newPassword": transactions.NewID()})
		r := httptest.NewRequest("POST", "http://app.test/api/v1/auth/reset-password", strings.NewReader(string(body)))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://app.test")
		r.Header.Set("Idempotency-Key", key)
		if csrf {
			r.Header.Set("X-CSRF-Token", envelope.Data.Token)
			for _, c := range bootstrap.Result().Cookies() {
				r.AddCookie(c)
			}
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := call(f.token, false, transactions.NewID()); w.Code != 403 {
		t.Fatalf("csrf: %d %s", w.Code, w.Body.String())
	}
	if w := call(f.token, true, ""); w.Code != 422 {
		t.Fatalf("key: %d %s", w.Code, w.Body.String())
	}
	if w := call(f.token, true, transactions.NewID()); w.Code != 200 || !strings.Contains(w.Body.String(), `"signInRequired":true`) {
		t.Fatalf("reset: %d %s", w.Code, w.Body.String())
	}
	unknown := strings.Repeat("z", 43)
	for i := 0; i < 5; i++ {
		if w := call(unknown, true, transactions.NewID()); w.Code != 422 {
			t.Fatalf("attempt: %d %s", w.Code, w.Body.String())
		}
	}
	if w := call(unknown, true, transactions.NewID()); w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatalf("limit: %d", w.Code)
	}
}
