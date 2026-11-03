package policy

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/testsupport"
)

type fixedClock struct{ now time.Time }

func (c *fixedClock) Now() time.Time { return c.now }

func exec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T, role string) (Service, context.Context, *fixedClock) {
	t.Helper()
	db := testsupport.Database(t)
	c := &fixedClock{time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)}
	exec(t, db, `INSERT INTO users(id,display_name,email,normalized_email,role_code,status,setup_required) VALUES('u','User','u@example.test','u@example.test',?,'active',0)`, role)
	exec(t, db, `INSERT INTO sessions(id,token_digest,user_id,access_revision,credential_revision,created_at,last_interaction_at,idle_expires_at,absolute_expires_at) VALUES('s','digest','u',1,1,?,?,?,?)`, c.now.Format(time.RFC3339Nano), c.now.Format(time.RFC3339Nano), c.now.Add(30*time.Minute).Format(time.RFC3339Nano), c.now.Add(12*time.Hour).Format(time.RFC3339Nano))
	for _, id := range []string{"p", "other"} {
		exec(t, db, `INSERT INTO projects(id,code,name,address,project_type,start_date,end_date,manager_id,created_by) VALUES(?,?,?,'address','building','2026-01-01','2026-12-31','u','u')`, id, id, id)
	}
	exec(t, db, `INSERT INTO user_projects(user_id,project_id,created_by) VALUES('u','p','u')`)
	return Service{DB: db, Clock: c}, WithPrincipal(context.Background(), Principal{UserID: "u", SessionDigest: "digest"}), c
}

func wantCode(t *testing.T, err error, status int, code string) {
	t.Helper()
	var safe *transactions.Error
	if !errors.As(err, &safe) || safe.Status != status || safe.Code != code {
		t.Fatalf("want %d %s, got %v", status, code, err)
	}
}

func TestEveryRoleAndDenyDefault(t *testing.T) {
	cases := []struct{ role, landing, allowed, denied string }{
		{"administrator", "/dashboard", "users.manage", "madeUp.manage"},
		{"project-manager", "/projects", "purchaseOrders.approve", "users.manage"},
		{"site-engineer", "/site-logs", "tasks.progress", "budgets.read"},
		{"procurement-officer", "/purchase-orders", "inventory.transfer", "purchaseOrders.approve"},
		{"finance-officer", "/invoices", "invoices.post", "refunds.approve"},
		{"sales-executive", "/leads", "bookings.collections", "payments.post"},
		{"management-viewer", "/dashboard", "reports.summary", "projects.read"},
	}
	for _, tc := range cases {
		t.Run(tc.role, func(t *testing.T) {
			s, ctx, _ := fixture(t, tc.role)
			if LandingPath(tc.role) != tc.landing {
				t.Fatal("incorrect landing")
			}
			if err := s.Authorize(ctx, tc.allowed, "p"); err != nil {
				t.Fatal(err)
			}
			wantCode(t, s.Authorize(ctx, tc.denied, "p"), 403, "FORBIDDEN")
			wantCode(t, s.Authorize(ctx, "*", "p"), 403, "FORBIDDEN")
			if tc.role == "administrator" {
				if err := s.Authorize(ctx, tc.allowed, "other"); err != nil {
					t.Fatal(err)
				}
			} else {
				wantCode(t, s.Authorize(ctx, tc.allowed, "other"), 403, "FORBIDDEN")
			}
			wantCode(t, s.Authorize(ctx, tc.allowed, "missing"), 403, "FORBIDDEN")
			grants := Capabilities(tc.role)
			grants[0] = "forged"
			if hasCapability(tc.role, "forged") {
				t.Fatal("grant registry escaped")
			}
		})
	}
	for _, role := range []string{"", "Administrator", "administrator,*", "unknown"} {
		if len(Capabilities(role)) != 0 || LandingPath(role) != "/login" {
			t.Fatal("unknown role granted")
		}
	}
}

func TestSessionExpiryAndClaims(t *testing.T) {
	s, ctx, c := fixture(t, "administrator")
	wantCode(t, s.Authorize(context.Background(), "users.manage", ""), 401, "AUTH_REQUIRED")
	forged := context.WithValue(context.Background(), "role", "administrator")
	wantCode(t, s.Authorize(forged, "users.manage", ""), 401, "AUTH_REQUIRED")
	wrong := WithPrincipal(ctx, Principal{UserID: "somebody-else", SessionDigest: "digest"})
	wantCode(t, s.Authorize(wrong, "users.manage", ""), 401, "SESSION_EXPIRED")
	c.now = c.now.Add(30 * time.Minute)
	wantCode(t, s.Authorize(ctx, "users.manage", ""), 401, "SESSION_EXPIRED")
	var last string
	if err := s.DB.QueryRow(`SELECT last_interaction_at FROM sessions WHERE id='s'`).Scan(&last); err != nil {
		t.Fatal(err)
	}
	if last != "2026-11-03T10:00:00Z" {
		t.Fatal("policy refreshed idle")
	}
}

func TestTransactionRechecksAllRevocations(t *testing.T) {
	changes := []struct {
		name, sql, code string
		status          int
	}{
		{"revoked", `UPDATE sessions SET revoked_at='2026-11-03T10:00:00Z' WHERE id='s' AND token_digest='digest' AND user_id='u'`, "SESSION_EXPIRED", 401},
		{"access revision", `UPDATE users SET access_revision=access_revision+1 WHERE id='u'`, "SESSION_EXPIRED", 401},
		{"password revision", `UPDATE users SET version=version+1 WHERE id='u'`, "SESSION_EXPIRED", 401},
		{"deactivated", `UPDATE users SET status='deactivated' WHERE id='u'`, "SESSION_EXPIRED", 401},
		{"setup", `UPDATE users SET setup_required=1 WHERE id='u'`, "SESSION_EXPIRED", 401},
		{"role", `UPDATE users SET role_code='management-viewer' WHERE id='u'`, "FORBIDDEN", 403},
		{"assignment", `DELETE FROM user_projects WHERE user_id='u' AND project_id='p'`, "FORBIDDEN", 403},
		{"absolute expiry", `UPDATE sessions SET idle_expires_at='2026-11-03T10:00:00Z',absolute_expires_at='2026-11-03T10:00:00Z',created_at='2026-11-03T09:00:00Z' WHERE id='s' AND token_digest='digest' AND user_id='u'`, "SESSION_EXPIRED", 401},
	}
	for _, tc := range changes {
		t.Run(tc.name, func(t *testing.T) {
			s, ctx, _ := fixture(t, "project-manager")
			if err := s.Authorize(ctx, "projects.manage", "p"); err != nil {
				t.Fatal(err)
			}
			tx, err := s.DB.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if _, err = tx.ExecContext(ctx, tc.sql); err != nil {
				t.Fatal(err)
			}
			// A nil DB proves the supplied transaction is the only database used.
			s.DB = nil
			wantCode(t, s.AuthorizeTx(ctx, tx, "projects.manage", "p"), tc.status, tc.code)
		})
	}
}

func TestScopedQueriesAndProjection(t *testing.T) {
	s, ctx, _ := fixture(t, "sales-executive")
	exec(t, s.DB, `INSERT INTO users(id,display_name,email,normalized_email,role_code) VALUES('v','Other','v@example.test','v@example.test','sales-executive')`)
	for _, row := range []struct{ id, project, owner string }{{"own", "p", "u"}, {"unassigned", "p", "v"}, {"wrong-project", "other", "u"}} {
		exec(t, s.DB, `INSERT INTO leads(id,project_id,contact_name,source,sales_executive_id,notes,last_activity_at,created_by) VALUES(?,?,'Contact','referral',?,'private','2026-11-03T10:00:00Z','u')`, row.id, row.project, row.owner)
	}
	scope, err := s.Scope(ctx, "leads")
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err = s.DB.QueryRow("SELECT count(*) FROM "+scope.Table+" WHERE ("+scope.Predicate+") AND r.id IN ('own','unassigned','wrong-project')", scope.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("scoped count %d", count)
	}
	rows, err := s.DB.Query("SELECT "+scope.Columns+" FROM "+scope.Table+" WHERE ("+scope.Predicate+") AND r.id IN ('own','unassigned','wrong-project')", scope.Args...)
	if err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if strings.Contains(scope.Columns, "*") || strings.Contains(scope.Columns, "notes") {
		t.Fatal("unsafe projection")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = s.RequireResourceTx(ctx, tx, "leads", "own"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"unassigned", "wrong-project", "missing", "' OR 1=1 --"} {
		wantCode(t, s.RequireResourceTx(ctx, tx, "leads", id), 404, "NOT_FOUND")
	}
	wantCode(t, s.RequireResourceTx(ctx, tx, "invoices", "any"), 404, "NOT_FOUND")
}

func TestRoutesAreClosedAndReadGrantsSeparate(t *testing.T) {
	for resource, grants := range routeResources {
		for _, suffix := range []string{"", "/id"} {
			cap, individual, known := RouteCapability("GET", "/api/v1/"+resource+suffix)
			if !known || cap != grants[0] || individual != (suffix != "") {
				t.Fatalf("route %s%s", resource, suffix)
			}
			if strings.HasSuffix(cap, ".manage") {
				t.Fatal("GET uses management grant")
			}
			if !hasCapability("administrator", cap) {
				t.Fatalf("unregistered capability %s", cap)
			}
		}
	}
	for _, path := range []string{"/api/v1/reports/unknown", "/api/v1/reports/receivables", "/api/v1/projects/id/unknown", "/api/v1/invoices/id/post", "/api/v1/projects/../users", "/api/v1/projects%2fusers", "/api/v1/unknown"} {
		if _, _, known := RouteCapability("GET", path); known {
			t.Fatalf("unknown path granted: %s", path)
		}
	}
	for _, method := range []string{"DELETE", "PATCH", "TRACE", ""} {
		if _, _, known := RouteCapability(method, "/api/v1/projects"); known {
			t.Fatal("unknown method granted")
		}
	}
}
