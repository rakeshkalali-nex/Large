package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/testsupport"
	fixtures "buildestate/backend/internal/testsupport/transactions"
)

func stewardExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}
func stewardDate(v string) time.Time {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		panic(err)
	}
	return t
}

func TestStewardLinkedPaymentHoldProtectsCustomerAndSourceLead(t *testing.T) {
	db := testsupport.Database(t)
	f := fixtures.Seed(t, db)
	ctx := context.Background()
	stewardExec(t, db, `INSERT INTO leads(id,project_id,contact_name,source,sales_executive_id,notes,last_activity_at,created_by) VALUES('source',?,'Buyer','phone',?,'','2020-01-01T00:00:00Z',?)`, f.Project, f.Creator, f.Creator)
	stewardExec(t, db, `INSERT INTO customers(id,project_id,name,source_lead_id,created_by) VALUES('buyer',?,'Buyer','source',?)`, f.Project, f.Creator)
	stewardExec(t, db, `INSERT INTO customer_contacts(id,customer_id,name,email) VALUES('contact','buyer','Buyer','original@example.invalid')`)
	stewardExec(t, db, `INSERT INTO payments(id,project_id,direction,counterparty_type,customer_id,amount_minor,method,reference,date,notes,created_by) VALUES('payment',?,'incoming','customer','buyer',100,'cash','Ref','2026-01-01','',?)`, f.Project, f.Creator)
	stewardExec(t, db, `INSERT INTO retention_holds(id,record_type,record_id,reason,created_by) VALUES('hold','payment','payment','Evidence preservation',?)`, f.Creator)
	for _, subject := range []struct{ kind, id string }{{"customer", "buyer"}, {"lead", "source"}} {
		err := (transactions.Manager{DB: db}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
			return stewardContact(ctx, tx, subject.kind, subject.id, stewardCorrections{}, true, time.Now())
		})
		var conflict *transactions.Error
		if !errors.As(err, &conflict) || conflict.Code != "RETENTION_HOLD" {
			t.Fatalf("%s bypassed source hold: %v", subject.kind, err)
		}
	}
	var email string
	if err := db.QueryRow(`SELECT email FROM customer_contacts WHERE id='contact'`).Scan(&email); err != nil {
		t.Fatal(err)
	}
	if email != "original@example.invalid" {
		t.Fatal("held contact changed")
	}
}

func TestStewardHTTPPrivacyReviewExecuteAuditAndMissing(t *testing.T) {
	s, ctx := settingsFixture(t)
	mux := http.NewServeMux()
	s.registerStewardship(mux)
	call := func(path, body string, want int) map[string]any {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)).WithContext(ctx)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", transactions.NewID())
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		var v struct {
			Data map[string]any `json:"data"`
		}
		if want == 200 {
			if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
				t.Fatal(err)
			}
		}
		return v.Data
	}
	v := call("/api/v1/privacy-requests", `{"subjectType":"user","subjectId":"admin","kind":"correction","reason":"Correct contact"}`, 200)
	id := v["id"].(string)
	call("/api/v1/privacy-requests/"+id+"/review", `{"version":1,"decision":"approve","reason":"Verified","corrections":{"name":"Corrected Admin"}}`, 200)
	call("/api/v1/privacy-requests/"+id+"/execute", `{"version":1}`, 409)
	stewardExec(t, s.D.DB, `CREATE TRIGGER steward_fail_audit BEFORE INSERT ON audit_events WHEN NEW.action='privacy-request.execute' BEGIN SELECT RAISE(ABORT,'test fault'); END`)
	call("/api/v1/privacy-requests/"+id+"/execute", `{"version":2}`, 500)
	var name, status string
	if err := s.D.DB.QueryRow(`SELECT display_name FROM users WHERE id='admin'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Admin" {
		t.Fatal("audit failure leaked contact update")
	}
	stewardExec(t, s.D.DB, `DROP TRIGGER steward_fail_audit`)
	call("/api/v1/privacy-requests/missing/review", `{"version":1,"decision":"approve","reason":"Verified"}`, 404)
	call("/api/v1/privacy-requests/missing/execute", `{"version":1}`, 404)
	call("/api/v1/privacy-requests/"+id+"/execute", `{"version":2}`, 200)
	if err := s.D.DB.QueryRow(`SELECT status FROM privacy_requests WHERE id=?`, id).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "completed" {
		t.Fatal(status)
	}
	var n int
	if err := s.D.DB.QueryRow(`SELECT count(*) FROM audit_events WHERE record_id=?`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("want three atomic audit events, got %d", n)
	}
	var revision int
	if err := s.D.DB.QueryRow(`SELECT access_revision FROM users WHERE id='admin'`).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	if revision != 2 {
		t.Fatalf("identity revision not advanced: %d", revision)
	}
}

func TestStewardListsAreBounded(t *testing.T) {
	db := testsupport.Database(t)
	f := fixtures.Seed(t, db)
	stewardExec(t, db, `WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<205) INSERT INTO privacy_requests(id,subject_type,subject_id,kind,reason,created_by) SELECT 'request-'||x,'user',?,'access','Access',? FROM n`, f.Creator, f.Creator)
	v, err := stewardRead(context.Background(), db, "privacy-requests", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(v.([]json.RawMessage)) != 100 {
		t.Fatal("unbounded list")
	}
	v, err = stewardRead(context.Background(), db, "privacy-requests", "", stewardPage{Limit: 10, Offset: 200})
	if err != nil {
		t.Fatal(err)
	}
	if len(v.([]json.RawMessage)) != 5 {
		t.Fatal("offset not applied")
	}
}

func TestStewardPagingRejectsUnknownAndInvalidQueries(t *testing.T) {
	for _, raw := range []string{"search=x", "limit=101", "limit=0", "offset=-1", "limit=1&limit=2", "offset=no", "limit=%zz"} {
		if _, err := stewardPaging(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	p, err := stewardPaging("limit=25&offset=50")
	if err != nil || p.Limit != 25 || p.Offset != 50 {
		t.Fatalf("paging %+v %v", p, err)
	}
	s, ctx := settingsFixture(t)
	mux := http.NewServeMux()
	s.registerStewardship(mux)
	for _, resource := range []string{"privacy-requests", "retention-holds", "retention-reviews"} {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/"+resource+"?unknown=1", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 422 {
			t.Fatalf("%s accepted unknown query: %d", resource, w.Code)
		}
	}
}

func TestStewardRetentionPeriodsAndEvidence(t *testing.T) {
	db := testsupport.Database(t)
	f := fixtures.Seed(t, db)
	ctx := context.Background()
	stewardExec(t, db, `UPDATE projects SET status='completed',closed_at='2020-01-01T00:00:00Z' WHERE id=?`, f.Project)
	stewardExec(t, db, `UPDATE users SET status='deactivated',deactivated_at='2026-01-01T00:00:00Z' WHERE id=?`, f.Reviewer)
	stewardExec(t, db, `INSERT INTO leads(id,project_id,contact_name,contact_email,normalized_email,source,sales_executive_id,notes,last_activity_at,created_by) VALUES('lead',?,'Buyer','buyer@example.invalid','buyer@example.invalid','phone',?,'Retained evidence','2024-01-01T00:00:00Z',?)`, f.Project, f.Creator, f.Creator)
	stewardExec(t, db, `INSERT INTO sessions(id,token_digest,user_id,access_revision,created_at,last_interaction_at,idle_expires_at,absolute_expires_at) VALUES('session','digest',?,1,'2025-12-01T00:00:00Z','2025-12-01T00:00:00Z','2026-01-01T00:00:00Z','2026-01-02T00:00:00Z')`, f.Reviewer)
	stewardExec(t, db, `INSERT INTO password_tokens(id,token_digest,user_id,kind,issuer_id,created_at,expires_at,reason) VALUES('token','token-digest',?,'setup',?,'2025-12-01T00:00:00Z','2026-01-01T00:00:00Z','Setup')`, f.Reviewer, f.Creator)
	for _, class := range []string{"authentication", "business"} {
		stewardExec(t, db, `INSERT INTO audit_events(id,actor_id,occurred_at,action,project_id,record_type,record_id,request_id,changes_json,retention_class) VALUES(?,?,'2025-01-01T00:00:00Z','test',?,'project',?,'request','[]',?)`, class, f.Creator, f.Project, f.Project, class)
	}
	for _, tc := range []struct{ kind, id, at, action string }{
		{"lead", "lead", "2026-01-01T00:00:00Z", "anonymize"},
		{"user", f.Reviewer, "2026-04-01T00:00:00Z", "anonymize"},
		{"session", "session", "2026-01-31T00:00:00Z", "purge"},
		{"password-token", "token", "2026-01-31T00:00:00Z", "purge"},
		{"project", f.Project, "2027-01-01T00:00:00Z", "retain"},
		{"audit-event", "authentication", "2026-01-01T00:00:00Z", "retain"},
		{"audit-event", "business", "2027-01-01T00:00:00Z", "retain"},
	} {
		t.Run(tc.kind+tc.id, func(t *testing.T) {
			at, action, err := stewardEligible(ctx, db, tc.kind, tc.id)
			if err != nil || !at.Equal(stewardDate(tc.at)) || action != tc.action {
				t.Fatalf("at=%v action=%s err=%v", at, action, err)
			}
		})
	}
	before, err := stewardCandidates(ctx, db, stewardDate("2025-12-31T23:59:59Z"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 0 {
		t.Fatalf("premature candidates: %+v", before)
	}
	if err = stewardContact(ctx, db, "lead", "lead", stewardCorrections{}, true, stewardDate("2026-01-02T00:00:00Z")); err != nil {
		t.Fatal(err)
	}
	var name, email, notes string
	if err = db.QueryRow(`SELECT contact_name,contact_email,notes FROM leads WHERE id='lead'`).Scan(&name, &email, &notes); err != nil {
		t.Fatal(err)
	}
	if name != "Retained contact" || email != "" || notes != "Retained evidence" {
		t.Fatalf("unexpected retained lead %q %q %q", name, email, notes)
	}
	if _, err = db.Exec(`DELETE FROM audit_events WHERE id='authentication'`); err == nil {
		t.Fatal("immutable authentication evidence was deleted")
	}
}

func TestStewardDirectAndProjectHoldsBlockContactChanges(t *testing.T) {
	for _, holdKind := range []string{"lead", "project"} {
		t.Run(holdKind, func(t *testing.T) {
			db := testsupport.Database(t)
			f := fixtures.Seed(t, db)
			ctx := context.Background()
			stewardExec(t, db, `INSERT INTO leads(id,project_id,contact_name,contact_email,source,sales_executive_id,notes,last_activity_at,created_by) VALUES('lead',?,'Buyer','original@example.invalid','phone',?,'Evidence','2020-01-01T00:00:00Z',?)`, f.Project, f.Creator, f.Creator)
			target := "lead"
			if holdKind == "project" {
				target = f.Project
			}
			stewardExec(t, db, `INSERT INTO retention_holds(id,record_type,record_id,reason,created_by) VALUES('hold',?,?,'Legal hold',?)`, holdKind, target, f.Creator)
			err := (transactions.Manager{DB: db}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
				return stewardContact(ctx, tx, "lead", "lead", stewardCorrections{}, true, time.Now())
			})
			var conflict *transactions.Error
			if !errors.As(err, &conflict) || conflict.Code != "RETENTION_HOLD" {
				t.Fatalf("wanted hold conflict: %v", err)
			}
			candidates, err := stewardCandidates(ctx, db, stewardDate("2026-01-01T00:00:00Z"), f.Project)
			if err != nil {
				t.Fatal(err)
			}
			if len(candidates) != 1 || candidates[0].Action != "retain" || len(candidates[0].HoldIDs) != 1 {
				t.Fatalf("held candidates: %+v", candidates)
			}
			var email string
			if err = db.QueryRow(`SELECT contact_email FROM leads WHERE id='lead'`).Scan(&email); err != nil {
				t.Fatal(err)
			}
			if email != "original@example.invalid" {
				t.Fatal("held contact changed")
			}
		})
	}
}

func TestStewardConvertedLeadAndActiveStaffArePreserved(t *testing.T) {
	db := testsupport.Database(t)
	f := fixtures.Seed(t, db)
	ctx := context.Background()
	stewardExec(t, db, `INSERT INTO leads(id,project_id,contact_name,source,sales_executive_id,notes,last_activity_at,created_by) VALUES('lead',?,'Buyer','phone',?,'','2020-01-01T00:00:00Z',?)`, f.Project, f.Creator, f.Creator)
	stewardExec(t, db, `INSERT INTO customers(id,project_id,name,source_lead_id,created_by) VALUES('customer',?,'Buyer','lead',?)`, f.Project, f.Creator)
	if _, _, err := stewardEligible(ctx, db, "lead", "lead"); err != sql.ErrNoRows {
		t.Fatalf("converted lead eligible: %v", err)
	}
	if err := stewardContact(ctx, db, "user", f.Creator, stewardCorrections{}, true, time.Now()); err == nil {
		t.Fatal("active staff anonymized")
	}
}

func TestStewardContactTransactionRollsBackAndNormalizes(t *testing.T) {
	db := testsupport.Database(t)
	f := fixtures.Seed(t, db)
	ctx := context.Background()
	name, email := "Corrected", "New@example.invalid"
	sentinel := errors.New("audit failed")
	err := (transactions.Manager{DB: db}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		if e := stewardContact(ctx, tx, "user", f.Creator, stewardCorrections{Name: &name, Email: &email}, false, time.Now()); e != nil {
			return e
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	var got string
	if err = db.QueryRow(`SELECT display_name FROM users WHERE id=?`, f.Creator).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "Test" {
		t.Fatal("contact change escaped rollback")
	}
	if err = stewardContact(ctx, db, "user", f.Creator, stewardCorrections{Name: &name, Email: &email}, false, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT normalized_email FROM users WHERE id=?`, f.Creator).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "new@example.invalid" {
		t.Fatal(got)
	}
}

func TestStewardPrivacyAccessProjectionAndEmptyLists(t *testing.T) {
	db := testsupport.Database(t)
	f := fixtures.Seed(t, db)
	ctx := context.Background()
	for _, resource := range []string{"privacy-requests", "retention-holds", "retention-reviews"} {
		v, err := stewardRead(ctx, db, resource, "")
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(v)
		if err != nil || string(raw) != "[]" {
			t.Fatalf("%s empty list %s %v", resource, raw, err)
		}
	}
	stewardExec(t, db, `INSERT INTO privacy_requests(id,subject_type,subject_id,kind,reason,status,reviewed_by,reviewed_at,review_reason,completed_at,created_by) VALUES('access','user',?,'access','Subject access','completed',?,'2026-01-01T00:00:00Z','Approved','2026-01-01T00:00:00Z',?)`, f.Creator, f.Reviewer, f.Creator)
	v, err := stewardRead(ctx, db, "privacy-requests", "access")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		AccessResult map[string]any `json:"accessResult"`
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.AccessResult) != 3 || result.AccessResult["name"] != "Test" || result.AccessResult["phone"] != nil {
		t.Fatalf("unexpected access projection %s", raw)
	}
	stewardExec(t, db, `UPDATE privacy_requests SET status='approved',completed_at=NULL WHERE id='access'`)
	v, err = stewardRead(ctx, db, "privacy-requests", "access")
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	result.AccessResult = nil
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.AccessResult != nil {
		t.Fatal("access details disclosed before execution")
	}
}
