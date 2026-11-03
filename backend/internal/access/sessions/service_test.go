package sessions_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/testsupport"
)

const sessionTimeFormat = "2006-01-02T15:04:05.000Z"

type sessionClock struct{ now time.Time }

func (c *sessionClock) Now() time.Time { return c.now }

type sessionFixture struct {
	db       *sql.DB
	clock    *sessionClock
	deps     contracts.Dependencies
	service  *sessions.Service
	password string
}

func newSessionFixture(t *testing.T, role string) *sessionFixture {
	t.Helper()
	db := testsupport.Database(t)
	clock := &sessionClock{now: time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)}
	key, secret := make([]byte, 32), make([]byte, 24)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	password := base64.RawURLEncoding.EncodeToString(secret)
	hash, err := sessions.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatal("fixture requires real Argon2id credentials")
	}
	_, err = db.Exec(`INSERT INTO users(id,display_name,email,normalized_email,role_code,status,password_hash,setup_required) VALUES('staff','Session Staff','staff@example.test','staff@example.test',?,'active',?,0)`, role, hash)
	if err != nil {
		t.Fatal(err)
	}
	deps := contracts.Dependencies{DB: db, Clock: clock, SessionKey: key}
	return &sessionFixture{db: db, clock: clock, deps: deps, service: sessions.New(deps), password: password}
}

func (f *sessionFixture) login(t *testing.T, prior string) (string, sessions.Identity) {
	t.Helper()
	token, identity, err := f.service.Login(context.Background(), " STAFF@EXAMPLE.TEST ", f.password, prior)
	if err != nil {
		t.Fatal(err)
	}
	return token, identity
}

func requireSessionError(t *testing.T, err error, status int, code string) {
	t.Helper()
	var safe *transactions.Error
	if !errors.As(err, &safe) || safe.Status != status || safe.Code != code {
		t.Fatalf("expected %d %s, got %v", status, code, err)
	}
}

func (f *sessionFixture) deadlines(t *testing.T, token string) (string, string, string) {
	t.Helper()
	var last, idle, absolute string
	err := f.db.QueryRow(`SELECT last_interaction_at,idle_expires_at,absolute_expires_at FROM sessions WHERE token_digest=?`, sessions.Digest(token)).Scan(&last, &idle, &absolute)
	if err != nil {
		t.Fatal(err)
	}
	return last, idle, absolute
}

func TestSessionLifecycleRotationHashedStorageAndLogout(t *testing.T) {
	f := newSessionFixture(t, "administrator")
	first, identity := f.login(t, "")
	raw, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil || len(raw) != 32 {
		t.Fatal("session must contain 32 random bytes")
	}
	if identity.User.ID != "staff" || identity.User.Role != "administrator" || identity.SetupRequired {
		t.Fatalf("unexpected identity: %+v", identity)
	}
	var digest, user string
	var access, credential int
	if err := f.db.QueryRow(`SELECT token_digest,user_id,access_revision,credential_revision FROM sessions WHERE user_id='staff' AND token_digest=?`, sessions.Digest(first)).Scan(&digest, &user, &access, &credential); err != nil {
		t.Fatal(err)
	}
	if digest != sessions.Digest(first) || digest == first || user != "staff" || access != 1 || credential != 1 {
		t.Fatal("incorrect persisted session metadata")
	}
	last, idle, absolute := f.deadlines(t, first)
	if last != f.clock.now.Format(sessionTimeFormat) || idle != f.clock.now.Add(30*time.Minute).Format(sessionTimeFormat) || absolute != f.clock.now.Add(12*time.Hour).Format(sessionTimeFormat) {
		t.Fatal("incorrect lifecycle deadlines")
	}
	second, _ := f.login(t, first)
	if first == second {
		t.Fatal("login did not rotate token")
	}
	_, err = f.service.Restore(context.Background(), first)
	requireSessionError(t, err, 401, "SESSION_EXPIRED")
	if _, err := sessions.New(f.deps).Restore(context.Background(), second); err != nil {
		t.Fatalf("new service cannot restore persisted session: %v", err)
	}
	if err := f.service.Logout(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if err := f.service.Logout(context.Background(), second); err != nil {
		t.Fatalf("logout must be idempotent: %v", err)
	}
	_, err = f.service.Restore(context.Background(), second)
	requireSessionError(t, err, 401, "SESSION_EXPIRED")
	var revoked int
	if err := f.db.QueryRow(`SELECT count(*) FROM sessions WHERE user_id='staff' AND token_digest IN (?,?) AND revoked_at IS NOT NULL`, sessions.Digest(first), sessions.Digest(second)).Scan(&revoked); err != nil {
		t.Fatal(err)
	}
	if revoked != 2 {
		t.Fatalf("revoked sessions = %d, want 2", revoked)
	}
}

func TestSessionRestoreDoesNotTouchIdleExpiry(t *testing.T) {
	f := newSessionFixture(t, "administrator")
	token, _ := f.login(t, "")
	last, idle, absolute := f.deadlines(t, token)
	f.clock.now = f.clock.now.Add(30*time.Minute - time.Millisecond)
	if _, err := f.service.Restore(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	a, b, c := f.deadlines(t, token)
	if a != last || b != idle || c != absolute {
		t.Fatal("identity restoration extended session")
	}
	f.clock.now = f.clock.now.Add(time.Millisecond)
	_, err := f.service.Restore(context.Background(), token)
	requireSessionError(t, err, 401, "SESSION_EXPIRED")
}

func TestAuthorizedInteractionsCannotExtendAbsoluteExpiry(t *testing.T) {
	f := newSessionFixture(t, "administrator")
	token, _ := f.login(t, "")
	start := f.clock.now
	noop := func(context.Context, contracts.Transaction) error { return nil }
	for elapsed := 20 * time.Minute; elapsed < 12*time.Hour; elapsed += 20 * time.Minute {
		f.clock.now = start.Add(elapsed)
		if err := f.service.WithinAuthorizedWrite(context.Background(), token, "users.manage", "", noop); err != nil {
			t.Fatalf("interaction at %s: %v", elapsed, err)
		}
	}
	_, idle, absolute := f.deadlines(t, token)
	want := start.Add(12 * time.Hour).Format(sessionTimeFormat)
	if idle != want || absolute != want {
		t.Fatalf("deadlines not capped: idle=%s absolute=%s", idle, absolute)
	}
	f.clock.now = start.Add(12 * time.Hour)
	called := false
	err := f.service.WithinAuthorizedWrite(context.Background(), token, "users.manage", "", func(context.Context, contracts.Transaction) error { called = true; return nil })
	requireSessionError(t, err, 401, "SESSION_EXPIRED")
	if called {
		t.Fatal("expired session reached mutation")
	}
}

func TestSessionCurrentAccountInvalidation(t *testing.T) {
	for _, tc := range []struct{ name, update string }{
		{"access revision", `UPDATE users SET access_revision=access_revision+1 WHERE id='staff'`},
		{"credential revision", `UPDATE users SET version=version+1 WHERE id='staff'`},
		{"deactivation", `UPDATE users SET status='deactivated' WHERE id='staff'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSessionFixture(t, "administrator")
			token, _ := f.login(t, "")
			if _, err := f.db.Exec(tc.update); err != nil {
				t.Fatal(err)
			}
			_, err := f.service.Restore(context.Background(), token)
			requireSessionError(t, err, 401, "SESSION_EXPIRED")
			called := false
			err = f.service.WithinAuthorizedWrite(context.Background(), token, "users.manage", "", func(context.Context, contracts.Transaction) error { called = true; return nil })
			requireSessionError(t, err, 401, "SESSION_EXPIRED")
			if called {
				t.Fatal("obsolete session reached write callback")
			}
		})
	}
}

func TestSetupRequiredSessionHasNoDomainGrants(t *testing.T) {
	f := newSessionFixture(t, "administrator")
	if _, err := f.db.Exec(`UPDATE users SET status='setup-required',setup_required=1 WHERE id='staff'`); err != nil {
		t.Fatal(err)
	}
	token, identity := f.login(t, "")
	if !identity.SetupRequired || len(identity.Capabilities) != 0 || len(identity.ProjectIDs) != 0 {
		t.Fatalf("setup identity exposes domain grants: %+v", identity)
	}
	called := false
	err := f.service.WithinAuthorizedWrite(context.Background(), token, "users.manage", "", func(context.Context, contracts.Transaction) error { called = true; return nil })
	if err == nil || called {
		t.Fatal("setup-required session was allowed a domain mutation")
	}
	if err := f.service.Logout(context.Background(), token); err != nil {
		t.Fatal(err)
	}
}

func TestAuthorizedWriteRollsBackBusinessEffectsAndIdleTouch(t *testing.T) {
	f := newSessionFixture(t, "administrator")
	if _, err := f.db.Exec(`CREATE TABLE session_test_effects (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	token, _ := f.login(t, "")
	last, idle, absolute := f.deadlines(t, token)
	f.clock.now = f.clock.now.Add(10 * time.Minute)
	injected := errors.New("injected mutation failure")
	err := f.service.WithinAuthorizedWrite(context.Background(), token, "users.manage", "", func(ctx context.Context, tx contracts.Transaction) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO session_test_effects VALUES('effect')`); err != nil {
			return err
		}
		return injected
	})
	if !errors.Is(err, injected) {
		t.Fatalf("callback error lost: %v", err)
	}
	var count int
	if err := f.db.QueryRow(`SELECT count(*) FROM session_test_effects WHERE id='effect'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed authorized mutation committed effects")
	}
	a, b, c := f.deadlines(t, token)
	if a != last || b != idle || c != absolute {
		t.Fatal("failed mutation committed idle touch")
	}
	called := false
	err = f.service.WithinAuthorizedWrite(context.Background(), token, "not.a.real.capability", "", func(context.Context, contracts.Transaction) error { called = true; return nil })
	requireSessionError(t, err, 403, "FORBIDDEN")
	if called {
		t.Fatal("unknown capability reached mutation")
	}
	a, b, c = f.deadlines(t, token)
	if a != last || b != idle || c != absolute {
		t.Fatal("denied authorization committed idle touch")
	}
	err = f.service.WithinAuthorizedWrite(context.Background(), token, "users.manage", "", func(ctx context.Context, tx contracts.Transaction) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO session_test_effects VALUES('committed')`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	a, b, c = f.deadlines(t, token)
	if a != f.clock.now.Format(sessionTimeFormat) || b != f.clock.now.Add(30*time.Minute).Format(sessionTimeFormat) || c != absolute {
		t.Fatal("successful authorized interaction did not refresh idle only")
	}
	if err := f.db.QueryRow(`SELECT count(*) FROM session_test_effects WHERE id='committed'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("successful mutation did not commit")
	}
}

func TestLoginFailuresDoNotRevealAccountState(t *testing.T) {
	f := newSessionFixture(t, "administrator")
	for _, tc := range []struct{ email, password string }{{"missing@example.test", f.password}, {"staff@example.test", "wrong-password"}} {
		_, _, err := f.service.Login(context.Background(), tc.email, tc.password, "")
		requireSessionError(t, err, 401, "INVALID_CREDENTIALS")
	}
	if _, err := f.db.Exec(`UPDATE users SET status='deactivated' WHERE id='staff'`); err != nil {
		t.Fatal(err)
	}
	_, _, err := f.service.Login(context.Background(), "staff@example.test", f.password, "")
	requireSessionError(t, err, 401, "INVALID_CREDENTIALS")
	var count int
	if err := f.db.QueryRow(`SELECT count(*) FROM sessions WHERE user_id='staff'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed credentials created sessions")
	}
}
