package users_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"buildestate/backend/internal/access/passwords"
	"buildestate/backend/internal/access/recovery"
	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/access/users"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/testsupport"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type fixture struct {
	db       *sql.DB
	d        contracts.Dependencies
	s        *users.Service
	admin    string
	password string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testsupport.Database(t)
	d := contracts.Dependencies{DB: db, Clock: fixedClock{time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)}, SessionKey: []byte(strings.Repeat("k", 32))}
	f := &fixture{db: db, d: d, s: users.New(d), password: transactions.NewID()}
	f.admin = f.addActive(t, "admin", "administrator")
	return f
}

func (f *fixture) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) addActive(t *testing.T, id, role string) string {
	t.Helper()
	hash, err := sessions.HashPassword(f.password)
	if err != nil {
		t.Fatal(err)
	}
	f.exec(t, `INSERT INTO users(id,display_name,email,normalized_email,role_code,status,setup_required,password_hash,created_by) VALUES(?,?,?,?,?,'active',0,?,?)`, id, id, id+"@test.invalid", id+"@test.invalid", role, hash, nullableCreator(id))
	return f.login(t, id)
}

func nullableCreator(id string) any {
	if id == "admin" {
		return nil
	}
	return "admin"
}

func (f *fixture) login(t *testing.T, id string) string {
	t.Helper()
	token, _, err := f.s.Sessions.Login(context.Background(), id+"@test.invalid", f.password, "")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func (f *fixture) project(t *testing.T, id, creator string) {
	t.Helper()
	f.exec(t, `INSERT INTO projects(id,code,name,address,project_type,start_date,end_date,manager_id,created_by) VALUES(?,?,?,'Test address','residential','2026-01-01','2026-12-31',?,?)`, id, id, id, creator, creator)
}

func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var problem *transactions.Error
	if !errors.As(err, &problem) || problem.Code != code {
		t.Fatalf("want %s; got %v", code, err)
	}
}

func input(id string) users.Input {
	return users.Input{DisplayName: id, Email: id + "@test.invalid", Role: "site-engineer", ProjectIDs: []string{}, Version: 1}
}

func TestCreateNormalizesAndRejectsInvalidAssignments(t *testing.T) {
	f := newFixture(t)
	f.project(t, "project", "admin")
	ctx := context.Background()
	in := input("staff")
	in.DisplayName, in.Email, in.ProjectIDs = "  Staff Member  ", "  Staff@Test.Invalid  ", []string{"project"}
	created, err := f.s.Save(ctx, f.admin, "", transactions.NewID(), in)
	if err != nil {
		t.Fatal(err)
	}
	if created.DisplayName != "Staff Member" || created.Email != "Staff@Test.Invalid" || created.Status != "setup-required" || created.Version != 1 || !reflect.DeepEqual(created.ProjectIDs, []string{"project"}) {
		t.Fatalf("unexpected created user: %+v", created)
	}
	var normalized, creator string
	if err = f.db.QueryRow(`SELECT normalized_email,created_by FROM users WHERE id=?`, created.ID).Scan(&normalized, &creator); err != nil {
		t.Fatal(err)
	}
	if normalized != "staff@test.invalid" || creator != "admin" {
		t.Fatalf("normalization/attribution: %q %q", normalized, creator)
	}
	_, err = f.s.Save(ctx, f.admin, "", transactions.NewID(), input("staff"))
	requireCode(t, err, "EMAIL_IN_USE")
	for _, tc := range []struct {
		name   string
		change func(*users.Input)
	}{
		{"role", func(in *users.Input) { in.Role = "owner" }},
		{"project", func(in *users.Input) { in.ProjectIDs = []string{"missing"} }},
		{"duplicate project", func(in *users.Input) { in.ProjectIDs = []string{"project", "project"} }},
		{"email", func(in *users.Input) { in.Email = "not-an-email" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := input("invalid")
			tc.change(&bad)
			_, err := f.s.Save(ctx, f.admin, "", transactions.NewID(), bad)
			requireCode(t, err, "VALIDATION_FAILED")
			var n int
			if err = f.db.QueryRow(`SELECT count(*) FROM users WHERE normalized_email=?`, strings.ToLower(bad.Email)).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != 0 {
				t.Fatal("invalid create persisted a user")
			}
		})
	}
}

func TestNonAdministratorsCannotListSaveOrChangeStatus(t *testing.T) {
	for _, role := range []string{"project-manager", "site-engineer", "procurement-officer", "finance-officer", "sales-executive", "management-viewer"} {
		t.Run(role, func(t *testing.T) {
			f := newFixture(t)
			token := f.addActive(t, "staff", role)
			ctx := context.Background()
			_, err := f.s.List(ctx, token, "")
			requireCode(t, err, "FORBIDDEN")
			for _, id := range []string{"", "admin"} {
				_, err = f.s.Save(ctx, token, id, transactions.NewID(), input("unauthorized"))
				requireCode(t, err, "FORBIDDEN")
			}
			for _, action := range []string{"deactivate", "reactivate"} {
				_, err = f.s.ChangeStatus(ctx, token, "admin", transactions.NewID(), action, users.Transition{Version: 1, Reason: "Unauthorized attempt"})
				requireCode(t, err, "FORBIDDEN")
			}
			var status, roleAfter string
			var version int
			if err = f.db.QueryRow(`SELECT status,role_code,version FROM users WHERE id='admin'`).Scan(&status, &roleAfter, &version); err != nil {
				t.Fatal(err)
			}
			if status != "active" || roleAfter != "administrator" || version != 1 {
				t.Fatal("denied mutation changed administrator")
			}
		})
	}
}

func TestAccessChangesRevokeOnlyTargetSessions(t *testing.T) {
	for _, change := range []string{"role", "projects", "deactivate"} {
		t.Run(change, func(t *testing.T) {
			f := newFixture(t)
			target := f.addActive(t, "target", "site-engineer")
			second := f.login(t, "target")
			other := f.addActive(t, "other", "site-engineer")
			f.project(t, "project", "admin")
			ctx := context.Background()
			var err error
			if change == "deactivate" {
				_, err = f.s.ChangeStatus(ctx, f.admin, "target", transactions.NewID(), change, users.Transition{Version: 1, Reason: "Staff departure"})
			} else {
				in := input("target")
				if change == "role" {
					in.Role = "management-viewer"
				} else {
					in.ProjectIDs = []string{"project"}
				}
				_, err = f.s.Save(ctx, f.admin, "target", transactions.NewID(), in)
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, token := range []string{target, second} {
				if _, err = f.s.Sessions.Restore(ctx, token); err == nil {
					t.Fatal("target session survived access change")
				}
				var revoked sql.NullString
				if err = f.db.QueryRow(`SELECT revoked_at FROM sessions WHERE token_digest=?`, sessions.Digest(token)).Scan(&revoked); err != nil {
					t.Fatal(err)
				}
				if !revoked.Valid {
					t.Fatal("target session not explicitly revoked")
				}
			}
			for _, token := range []string{f.admin, other} {
				if _, err = f.s.Sessions.Restore(ctx, token); err != nil {
					t.Fatalf("unrelated session revoked: %v", err)
				}
			}
		})
	}
}

func TestLifecyclePreservesAttributionAndRequiresReplacementSetup(t *testing.T) {
	f := newFixture(t)
	oldSession := f.addActive(t, "target", "site-engineer")
	f.project(t, "historical-project", "target")
	f.exec(t, `INSERT INTO user_projects(user_id,project_id,created_by) VALUES('target','historical-project','admin')`)
	ctx := context.Background()
	r := recovery.New(f.d)
	old, _, err := r.Issue(ctx, f.admin, "target", transactions.NewID(), recovery.Input{Version: 1, Purpose: "recovery", IdentityVerified: true, Reason: "Identity verified in person"})
	if err != nil {
		t.Fatal(err)
	}
	var auditID, auditActor, auditChanges string
	if err = f.db.QueryRow(`SELECT id,actor_id,changes_json FROM audit_events WHERE actor_id='target' ORDER BY id LIMIT 1`).Scan(&auditID, &auditActor, &auditChanges); err != nil {
		t.Fatal(err)
	}
	for i, action := range []string{"deactivate", "reactivate"} {
		u, err := f.s.ChangeStatus(ctx, f.admin, "target", transactions.NewID(), action, users.Transition{Version: int64(i + 1), Reason: "Employment lifecycle change"})
		if err != nil {
			t.Fatal(err)
		}
		wantStatus := "deactivated"
		verb := "user.deactivate"
		if action == "reactivate" {
			wantStatus, verb = "setup-required", "user.restore"
		}
		if u.ID != "target" || u.Status != wantStatus || u.Version != int64(i+2) || !reflect.DeepEqual(u.ProjectIDs, []string{"historical-project"}) {
			t.Fatalf("unexpected lifecycle result: %+v", u)
		}
		var actor, reason string
		if err = f.db.QueryRow(`SELECT actor_id,reason FROM audit_events WHERE record_type='user' AND record_id='target' AND action=?`, verb).Scan(&actor, &reason); err != nil {
			t.Fatal(err)
		}
		if actor != "admin" || reason != "Employment lifecycle change" {
			t.Fatalf("missing lifecycle audit attribution: %q %q", actor, reason)
		}
	}
	var creator, manager, userCreator, preservedActor, preservedChanges string
	if err = f.db.QueryRow(`SELECT created_by,manager_id FROM projects WHERE id='historical-project'`).Scan(&creator, &manager); err != nil {
		t.Fatal(err)
	}
	if err = f.db.QueryRow(`SELECT created_by FROM users WHERE id='target'`).Scan(&userCreator); err != nil {
		t.Fatal(err)
	}
	if err = f.db.QueryRow(`SELECT actor_id,changes_json FROM audit_events WHERE id=?`, auditID).Scan(&preservedActor, &preservedChanges); err != nil {
		t.Fatal(err)
	}
	if creator != "target" || manager != "target" || userCreator != "admin" || preservedActor != auditActor || preservedChanges != auditChanges {
		t.Fatal("lifecycle rewrote historical attribution")
	}
	if _, err = f.s.Sessions.Restore(ctx, oldSession); err == nil {
		t.Fatal("old session restored after reactivation")
	}
	_, _, err = f.s.Sessions.Login(ctx, "target@test.invalid", f.password, "")
	requireCode(t, err, "INVALID_CREDENTIALS")
	p := passwords.New(f.d)
	requireCode(t, p.Reset(ctx, strings.TrimPrefix(old.ResetLink, "/login#token="), transactions.NewID(), transactions.NewID()), "INVALID_RESET_TOKEN")
	replacement, _, err := r.Issue(ctx, f.admin, "target", transactions.NewID(), recovery.Input{Version: 3, Purpose: "setup", IdentityVerified: true, Reason: "Returning staff identity verified"})
	if err != nil {
		t.Fatal(err)
	}
	newPassword := transactions.NewID()
	if err = p.Reset(ctx, strings.TrimPrefix(replacement.ResetLink, "/login#token="), newPassword, transactions.NewID()); err != nil {
		t.Fatal(err)
	}
	if _, _, err = f.s.Sessions.Login(ctx, "target@test.invalid", newPassword, ""); err != nil {
		t.Fatalf("replacement setup did not restore access: %v", err)
	}
	_, _, err = f.s.Sessions.Login(ctx, "target@test.invalid", f.password, "")
	requireCode(t, err, "INVALID_CREDENTIALS")
}

func TestFinalActiveAdministratorCannotBeRemoved(t *testing.T) {
	for _, action := range []string{"demote", "deactivate"} {
		t.Run(action, func(t *testing.T) {
			f := newFixture(t)
			var err error
			if action == "demote" {
				_, err = f.s.Save(context.Background(), f.admin, "admin", transactions.NewID(), input("admin"))
			} else {
				_, err = f.s.ChangeStatus(context.Background(), f.admin, "admin", transactions.NewID(), action, users.Transition{Version: 1, Reason: "Staff departure"})
			}
			requireCode(t, err, "FINAL_ACTIVE_ADMINISTRATOR")
			if _, err = f.s.Sessions.Restore(context.Background(), f.admin); err != nil {
				t.Fatalf("rejected change revoked session: %v", err)
			}
		})
	}
}

func TestCompetingAdministratorSelfDemotionsLeaveExactlyOneAdministrator(t *testing.T) {
	f := newFixture(t)
	second := f.addActive(t, "second", "administrator")
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, actor := range []struct{ id, token string }{{"admin", f.admin}, {"second", second}} {
		go func(id, token string) {
			<-start
			_, err := f.s.Save(context.Background(), token, id, transactions.NewID(), input(id))
			results <- err
		}(actor.id, actor.token)
	}
	close(start)
	succeeded, rejected := 0, 0
	for i := 0; i < 2; i++ {
		if err := <-results; err == nil {
			succeeded++
		} else {
			requireCode(t, err, "FINAL_ACTIVE_ADMINISTRATOR")
			rejected++
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("successes=%d rejected=%d", succeeded, rejected)
	}
	var n int
	if err := f.db.QueryRow(`SELECT count(*) FROM users WHERE id IN ('admin','second') AND role_code='administrator' AND status='active' AND setup_required=0`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("remaining active fixture administrators=%d", n)
	}
}
