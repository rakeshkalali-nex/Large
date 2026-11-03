package history_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/audit/history"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/testsupport"
	fixtures "buildestate/backend/internal/testsupport/transactions"
)

type historyClock struct{ now time.Time }

func (c *historyClock) Now() time.Time { return c.now }

type historyFixture struct {
	db         *sql.DB
	s          history.Service
	ctx        context.Context
	clock      *historyClock
	own, other fixtures.Fixture
}

func historyExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func newHistoryFixture(t *testing.T, role string) historyFixture {
	t.Helper()
	db := testsupport.Database(t)
	f := historyFixture{db: db, own: fixtures.Seed(t, db), other: fixtures.Seed(t, db)}
	f.clock = &historyClock{time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)}
	historyExec(t, db, `UPDATE users SET role_code=? WHERE id=?`, role, f.own.Creator)
	historyExec(t, db, `INSERT INTO user_projects(user_id,project_id,created_by) VALUES(?,?,?)`, f.own.Creator, f.own.Project, f.own.Reviewer)
	f.addSession(t, "history-session", "history-digest", f.own.Creator)
	f.ctx = policy.WithPrincipal(context.Background(), policy.Principal{UserID: f.own.Creator, SessionDigest: "history-digest"})
	f.s = history.Service{D: contracts.Dependencies{DB: db, Clock: f.clock}}
	return f
}

func (f historyFixture) addSession(t *testing.T, id, digest, user string) {
	t.Helper()
	historyExec(t, f.db, `INSERT INTO sessions(id,token_digest,user_id,access_revision,credential_revision,created_at,last_interaction_at,idle_expires_at,absolute_expires_at) VALUES(?,?,?,1,1,?,?,?,?)`, id, digest, user, f.clock.now.Format(time.RFC3339Nano), f.clock.now.Format(time.RFC3339Nano), f.clock.now.Add(30*time.Minute).Format(time.RFC3339Nano), f.clock.now.Add(12*time.Hour).Format(time.RFC3339Nano))
}

// Direct inserts also exercise the read boundary against retained legacy data,
// rather than relying on the current writer to have already sanitized it.
func (f historyFixture) event(t *testing.T, id string, project any, kind, action, at, changes string, reason any) {
	t.Helper()
	historyExec(t, f.db, `INSERT INTO audit_events(id,actor_id,occurred_at,action,project_id,record_type,record_id,request_id,reason,changes_json,origin_digest,retention_class) VALUES(?,?,?,?,?,?,?,?,?,?,?,'business')`, id, f.own.Reviewer, at, action, project, kind, "record-"+id, "request-"+id, reason, changes, "private-origin-digest")
}

const historyAt = "2026-11-03T09:00:00.000Z"

func historyIDs(events []history.Event) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	return ids
}

func historyWantError(t *testing.T, err error, status int, code string) {
	t.Helper()
	var public *transactions.Error
	if !errors.As(err, &public) || public.Status != status || public.Code != code {
		t.Fatalf("want %d %s, got %v", status, code, err)
	}
}

func TestHistoryReadRoleAndProjectScope(t *testing.T) {
	for _, tc := range []struct {
		role string
		want []string
	}{
		{"administrator", []string{"f", "e", "d", "c", "b", "a"}},
		{"finance-officer", []string{"e", "a"}},
		{"project-manager", []string{"b"}},
	} {
		t.Run(tc.role, func(t *testing.T) {
			f := newHistoryFixture(t, tc.role)
			f.event(t, "a", f.own.Project, "invoice", "post", historyAt, "[]", nil)
			f.event(t, "b", f.own.Project, "site-log", "site-log.approve", historyAt, "[]", nil)
			f.event(t, "c", f.own.Project, "site-log", "update", historyAt, "[]", nil)
			f.event(t, "d", f.other.Project, "purchase-order", "approve", historyAt, "[]", nil)
			f.event(t, "e", f.own.Project, "purchase-order", "create", historyAt, "[]", nil)
			f.event(t, "f", nil, "user", "update", historyAt, "[]", nil)
			events, _, err := f.s.Read(f.ctx, url.Values{}, "")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(historyIDs(events), tc.want) {
				t.Fatalf("got %v, want %v", historyIDs(events), tc.want)
			}
			for _, id := range []string{"a", "b", "c", "d", "e", "f", "missing"} {
				events, _, err := f.s.Read(f.ctx, url.Values{}, id)
				allowed := false
				for _, want := range tc.want {
					allowed = allowed || want == id
				}
				if allowed {
					if err != nil || len(events) != 1 || events[0].ID != id {
						t.Fatalf("detail %s: %v %v", id, events, err)
					}
				} else {
					historyWantError(t, err, 404, "NOT_FOUND")
				}
			}
			if tc.role != "administrator" {
				events, _, err := f.s.Read(f.ctx, url.Values{"projectId": {f.other.Project}}, "")
				if err != nil || len(events) != 0 {
					t.Fatalf("cross-project filter leaked: %v %v", events, err)
				}
			}
		})
	}
}

func TestHistoryReadDeniesRolesAndExpiredIdentity(t *testing.T) {
	for _, role := range []string{"site-engineer", "procurement-officer", "sales-executive", "management-viewer"} {
		t.Run(role, func(t *testing.T) {
			f := newHistoryFixture(t, role)
			_, _, err := f.s.Read(f.ctx, url.Values{}, "")
			historyWantError(t, err, 403, "FORBIDDEN")
		})
	}
	f := newHistoryFixture(t, "administrator")
	_, _, err := f.s.Read(context.Background(), url.Values{}, "")
	historyWantError(t, err, 401, "AUTH_REQUIRED")
	f.clock.now = f.clock.now.Add(30 * time.Minute)
	_, _, err = f.s.Read(f.ctx, url.Values{}, "")
	historyWantError(t, err, 401, "SESSION_EXPIRED")
	var last string
	if err := f.db.QueryRow(`SELECT last_interaction_at FROM sessions WHERE id='history-session'`).Scan(&last); err != nil {
		t.Fatal(err)
	}
	if last != "2026-11-03T10:00:00Z" {
		t.Fatalf("read refreshed expired session: %s", last)
	}
}

func TestHistoryReadPaginationAndFilters(t *testing.T) {
	f := newHistoryFixture(t, "administrator")
	for _, id := range []string{"a", "b", "c"} {
		f.event(t, id, f.own.Project, "purchase-order", "approve", historyAt, "[]", nil)
	}
	f.event(t, "d", f.other.Project, "invoice", "post", "2026-11-03T09:01:00.000Z", "[]", nil)
	for _, tc := range []struct {
		direction string
		want      []string
	}{{"desc", []string{"d", "c", "b", "a"}}, {"asc", []string{"a", "b", "c", "d"}}} {
		t.Run(tc.direction, func(t *testing.T) {
			q := url.Values{"limit": {"2"}, "direction": {tc.direction}}
			var got []string
			for pageNumber := 0; pageNumber < 3; pageNumber++ {
				events, page, err := f.s.Read(f.ctx, q, "")
				if err != nil {
					t.Fatal(err)
				}
				if page.Limit != 2 || len(events) > 2 {
					t.Fatalf("unbounded page: %+v", page)
				}
				got = append(got, historyIDs(events)...)
				if page.NextCursor == nil {
					break
				}
				q.Set("cursor", *page.NextCursor)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("pages: %v, want %v", got, tc.want)
			}
		})
	}
	for _, q := range []url.Values{
		{"actorId": {f.own.Reviewer}, "projectId": {f.own.Project}, "recordType": {"purchase-order"}, "action": {"approve"}, "recordId": {"record-b"}},
		{"q": {"REQUEST-B"}},
		{"from": {historyAt}, "to": {historyAt}, "recordId": {"record-b"}},
	} {
		events, _, err := f.s.Read(f.ctx, q, "")
		if err != nil || !reflect.DeepEqual(historyIDs(events), []string{"b"}) {
			t.Fatalf("filter %v: %v %v", q, historyIDs(events), err)
		}
	}
	for _, q := range []url.Values{{"q": {"' OR 1=1 --"}}, {"q": {"%"}}, {"actorId": {f.other.Creator}}} {
		events, _, err := f.s.Read(f.ctx, q, "")
		if err != nil || len(events) != 0 {
			t.Fatalf("literal/no-match filter %v: %v %v", q, events, err)
		}
	}
}

func TestHistoryReadRejectsInvalidQueriesAndReboundCursors(t *testing.T) {
	f := newHistoryFixture(t, "administrator")
	f.event(t, "a", f.own.Project, "purchase-order", "approve", historyAt, "[]", nil)
	f.event(t, "b", f.own.Project, "purchase-order", "approve", historyAt, "[]", nil)
	for _, q := range []url.Values{
		{"limit": {"0"}}, {"limit": {"101"}}, {"limit": {"1", "2"}},
		{"sort": {"reason"}}, {"direction": {"desc; DROP TABLE users"}}, {"role": {"administrator"}},
		{"from": {"yesterday"}}, {"from": {"2026-11-03T10:00:00+01:00"}},
		{"from": {"2026-11-04T00:00:00Z"}, "to": {historyAt}},
		{"q": {strings.Repeat("x", 201)}}, {"cursor": {"not-a-cursor"}},
	} {
		_, _, err := f.s.Read(f.ctx, q, "")
		historyWantError(t, err, 422, "VALIDATION_FAILED")
	}
	_, page, err := f.s.Read(f.ctx, url.Values{"limit": {"1"}}, "")
	if err != nil || page.NextCursor == nil {
		t.Fatalf("first page: %+v %v", page, err)
	}
	for _, change := range []url.Values{{"limit": {"2"}}, {"direction": {"asc"}}, {"action": {"approve"}}, {"projectId": {f.own.Project}}} {
		q := url.Values{"limit": {"1"}, "cursor": {*page.NextCursor}}
		for key, value := range change {
			q[key] = value
		}
		_, _, err := f.s.Read(f.ctx, q, "")
		historyWantError(t, err, 422, "VALIDATION_FAILED")
	}
	q := url.Values{"limit": {"1"}, "cursor": {*page.NextCursor}}
	f.addSession(t, "second-session", "second-digest", f.own.Creator)
	f.addSession(t, "other-session", "other-digest", f.other.Creator)
	for _, principal := range []policy.Principal{{UserID: f.own.Creator, SessionDigest: "second-digest"}, {UserID: f.other.Creator, SessionDigest: "other-digest"}} {
		_, _, err := f.s.Read(policy.WithPrincipal(context.Background(), principal), q, "")
		historyWantError(t, err, 422, "VALIDATION_FAILED")
	}
	historyExec(t, f.db, `UPDATE users SET role_code='finance-officer' WHERE id=?`, f.own.Creator)
	_, _, err = f.s.Read(f.ctx, q, "")
	historyWantError(t, err, 422, "VALIDATION_FAILED")
}

func historyHTTP(f historyFixture, method, target string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	history.RegisterRoutes(mux, f.s.D)
	r := httptest.NewRequest(method, target, strings.NewReader(`{"reason":"overwrite"}`)).WithContext(f.ctx)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestHistoryHTTPRedactsSecretsAndProjectsSafeChanges(t *testing.T) {
	for _, role := range []string{"administrator", "finance-officer", "project-manager"} {
		t.Run(role, func(t *testing.T) {
			f := newHistoryFixture(t, role)
			changes := `[{"field":"status","before":"submitted","after":"approved"},{"field":"amount_minor","before":"100","after":"200"},{"field":"password","after":"hunter2"},{"field":"status","after":"secret-value"},{"field":"version","after":"sqlite failure"},{"field":"notes","after":"private customer text"}]`
			f.event(t, "redacted", f.own.Project, "purchase-order", "approve", historyAt, changes, "Authorization: Bearer credential")
			for _, path := range []string{"/api/v1/audit-events", "/api/v1/audit-events/redacted"} {
				w := historyHTTP(f, http.MethodGet, path)
				if w.Code != 200 {
					t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
				}
				if w.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("audit response must not be cached")
				}
				for _, secret := range []string{"hunter2", "secret-value", "sqlite failure", "private customer text", "Bearer credential", "private-origin-digest", "origin_digest", "changes_json"} {
					if strings.Contains(w.Body.String(), secret) {
						t.Fatalf("response leaked %q", secret)
					}
				}
				var envelope struct{ Data json.RawMessage }
				if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
					t.Fatal(err)
				}
				var events []history.Event
				if path == "/api/v1/audit-events" {
					if err := json.Unmarshal(envelope.Data, &events); err != nil {
						t.Fatal(err)
					}
				} else {
					var event history.Event
					if err := json.Unmarshal(envelope.Data, &event); err != nil {
						t.Fatal(err)
					}
					events = []history.Event{event}
				}
				wantChanges := 2
				if role == "project-manager" {
					wantChanges = 1
				}
				if len(events) != 1 {
					t.Fatalf("events: %v", events)
				}
				e := events[0]
				if e.ID != "redacted" || e.Reason != nil || len(e.Changes) != wantChanges || e.Changes[0].Field != "status" || e.Changes[0].After == nil || *e.Changes[0].After != "approved" {
					t.Fatalf("unsafe or missing projection: %+v", e)
				}
				if e.RecordLink == nil || *e.RecordLink != "/purchase-orders/record-redacted" {
					t.Fatalf("record link: %v", e.RecordLink)
				}
			}
		})
	}
}

func TestHistoryHTTPIsImmutableAndErrorsAreScoped(t *testing.T) {
	f := newHistoryFixture(t, "project-manager")
	f.event(t, "visible", f.own.Project, "purchase-order", "approve", historyAt, "[]", "Approved after review")
	f.event(t, "hidden", f.other.Project, "purchase-order", "approve", historyAt, "[]", nil)
	for _, path := range []string{"/api/v1/audit-events/hidden", "/api/v1/audit-events/missing"} {
		w := historyHTTP(f, http.MethodGet, path)
		if w.Code != 404 || !strings.Contains(w.Body.String(), "NOT_FOUND") {
			t.Fatalf("hidden detail: %d %s", w.Code, w.Body.String())
		}
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, path := range []string{"/api/v1/audit-events", "/api/v1/audit-events/visible"} {
			w := historyHTTP(f, method, path)
			if w.Code != http.StatusMethodNotAllowed {
				t.Fatalf("%s %s: %d", method, path, w.Code)
			}
		}
	}
	var count int
	var reason string
	if err := f.db.QueryRow(`SELECT count(*) FROM audit_events WHERE id IN ('visible','hidden')`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(`SELECT reason FROM audit_events WHERE id='visible'`).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if count != 2 || reason != "Approved after review" {
		t.Fatalf("HTTP mutation altered audit data: %d %s", count, reason)
	}
	f.clock.now = f.clock.now.Add(30 * time.Minute)
	w := historyHTTP(f, http.MethodGet, "/api/v1/audit-events")
	if w.Code != 401 || !strings.Contains(w.Body.String(), "SESSION_EXPIRED") {
		t.Fatalf("expired HTTP read: %d %s", w.Code, w.Body.String())
	}
}
