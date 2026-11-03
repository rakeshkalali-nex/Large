package history

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"

	"buildestate/backend/internal/testsupport"
)

func retentionExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func retentionDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testsupport.Database(t)
	retentionExec(t, db, `INSERT INTO users(id,display_name,email,normalized_email,role_code) VALUES('retention-user','Retention','retention@example.test','retention@example.test','administrator')`)
	retentionExec(t, db, `INSERT INTO projects(id,code,name,address,project_type,start_date,end_date,manager_id,created_by,status,closed_at) VALUES('retention-project','RET','Retention','Site','construction','2010-01-01','2019-12-31','retention-user','retention-user','completed','2020-06-15T10:30:00Z')`)
	return db
}

func retentionEvent(t *testing.T, db *sql.DB, id, action, class, kind, record string, project any) {
	t.Helper()
	retentionExec(t, db, `INSERT INTO audit_events(id,actor_id,occurred_at,action,project_id,record_type,record_id,request_id,changes_json,retention_class) VALUES(?,'retention-user','2019-01-02T03:04:05Z',?,?,?,?,'retention-request','[]',?)`, id, action, project, kind, record, class)
}

func TestRetentionClosureClock(t *testing.T) {
	db := retentionDB(t)
	retentionEvent(t, db, "business", "project.update", "business", "project", "retention-project", "retention-project")
	for _, tc := range []struct{ status, closed, want string }{
		{"completed", "2020-06-15T10:30:00Z", "2027-06-15T10:30:00Z"},
		{"archived", "2021-01-01", "2028-01-01T00:00:00Z"},
		{"active", "2020-06-15T10:30:00Z", ""},
		{"completed", "", ""},
	} {
		retentionExec(t, db, `UPDATE projects SET status=?,closed_at=NULLIF(?,'') WHERE id='retention-project'`, tc.status, tc.closed)
		got, err := retentionFor(context.Background(), db, "business")
		if err != nil {
			t.Fatal(err)
		}
		if got.Class != "business" || (tc.want == "" && got.EligibleAt != nil) || (tc.want != "" && (got.EligibleAt == nil || *got.EligibleAt != tc.want)) {
			t.Fatalf("%+v: got %+v", tc, got)
		}
	}
	retentionEvent(t, db, "global", "settings.update", "business", "settings", "company", nil)
	got, err := retentionFor(context.Background(), db, "global")
	if err != nil || got.EligibleAt != nil {
		t.Fatalf("unclosed global evidence: %+v %v", got, err)
	}
}

func TestRetentionRoutineAndImportantAuthentication(t *testing.T) {
	db := retentionDB(t)
	for _, action := range strings.Fields("auth.login.succeeded auth.login.failed auth.logout auth.session.expired auth.access.denied auth.password.changed auth.password.reset.requested auth.password.reset.completed auth.setup.completed auth.unknown") {
		retentionEvent(t, db, action, action, "authentication", "authentication", "retention-user", nil)
		got, err := retentionFor(context.Background(), db, action)
		if err != nil {
			t.Fatal(err)
		}
		important := strings.HasPrefix(action, "auth.password.") || action == "auth.setup.completed" || action == "auth.unknown"
		if important {
			if got.Class != "business" || got.EligibleAt != nil {
				t.Fatalf("important %s: %+v", action, got)
			}
		} else if got.Class != "authentication" || got.EligibleAt == nil || *got.EligibleAt != "2020-01-02T03:04:05Z" {
			t.Fatalf("routine %s: %+v", action, got)
		}
	}
	retentionEvent(t, db, "important-project", "auth.password.changed", "authentication", "authentication", "retention-user", "retention-project")
	got, err := retentionFor(context.Background(), db, "important-project")
	if err != nil || got.EligibleAt == nil || *got.EligibleAt != "2027-06-15T10:30:00Z" {
		t.Fatalf("important parent clock: %+v %v", got, err)
	}
}

func TestRetentionHoldsAndRetainedExportSources(t *testing.T) {
	db := retentionDB(t)
	retentionExec(t, db, `INSERT INTO upload_batches(id,project_id,status,file_count,total_bytes,created_by) VALUES('retention-batch','retention-project','committed',1,10,'retention-user')`)
	retentionExec(t, db, `INSERT INTO documents(id,project_id,upload_batch_id,storage_key,original_name,mime_type,size_bytes,sha256,classification,created_by) VALUES('retention-document','retention-project','retention-batch','retention-key','report.pdf','application/pdf',10,?,'financial','retention-user')`, strings.Repeat("a", 64))
	retentionExec(t, db, `INSERT INTO report_snapshots(id,project_id,report,view,generated_at,document_id,filters_json,totals_json,source_versions_json,created_by,created_at) VALUES('retention-snapshot','retention-project','budget','costs','2020-01-01T00:00:00Z','retention-document','{}','{}','[{"resource":"invoice","id":"source-invoice","version":1},{"recordType":"payment","recordId":"source-payment","version":1}]','retention-user','2020-01-01T00:00:00Z')`)
	retentionEvent(t, db, "export", "document.create", "business", "document", "retention-document", "retention-project")
	for _, hold := range [][3]string{{"event-hold", "audit-event", "export"}, {"record-hold", "document", "retention-document"}, {"project-hold", "project", "retention-project"}, {"source-hold", "invoice", "source-invoice"}, {"payment-hold", "payment", "source-payment"}, {"released-hold", "invoice", "source-invoice"}, {"unrelated-hold", "invoice", "unrelated"}} {
		status := "active"
		if hold[0] == "released-hold" {
			status = "released"
		}
		retentionExec(t, db, `INSERT INTO retention_holds(id,record_type,record_id,reason,status,created_by) VALUES(?,?,?,'Evidence hold',?,'retention-user')`, hold[0], hold[1], hold[2], status)
	}
	got, err := retentionFor(context.Background(), db, "export")
	want := []string{"event-hold", "payment-hold", "project-hold", "record-hold", "source-hold"}
	if err != nil || !reflect.DeepEqual(got.HoldIDs, want) || got.EligibleAt != nil {
		t.Fatalf("inherited holds: %+v %v", got, err)
	}
}

func TestRetentionDoesNotMutateOrDeleteEvidence(t *testing.T) {
	db := retentionDB(t)
	retentionEvent(t, db, "immutable", "auth.password.changed", "authentication", "authentication", "retention-user", nil)
	if _, err := retentionFor(context.Background(), db, "immutable"); err != nil {
		t.Fatal(err)
	}
	var class string
	if err := db.QueryRow(`SELECT retention_class FROM audit_events WHERE id='immutable'`).Scan(&class); err != nil || class != "authentication" {
		t.Fatalf("projection changed stored evidence: %s %v", class, err)
	}
	for _, statement := range []string{`UPDATE audit_events SET retention_class='business' WHERE id='immutable'`, `DELETE FROM audit_events WHERE id='immutable'`} {
		if _, err := db.Exec(statement); err == nil {
			t.Fatal("immutable row accepted mutation")
		}
	}
	if _, err := retentionFor(context.Background(), db, "absent"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing event: %v", err)
	}
}
