// Package transactions provides real retained records for infrastructure tests.
package transactions

import (
	"buildestate/backend/internal/platform/transactions"
	"database/sql"
	"testing"
)

type Fixture struct{ Creator, Reviewer, Project, Site, Log string }

func Seed(t testing.TB, db *sql.DB) Fixture {
	t.Helper()
	f := Fixture{transactions.NewID(), transactions.NewID(), transactions.NewID(), transactions.NewID(), transactions.NewID()}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{f.Creator, f.Reviewer} {
		exec(`INSERT INTO users(id,display_name,email,normalized_email,role_code,status,setup_required) VALUES(?,?,?,?,'administrator','active',0)`, id, "Test", id+"@example.invalid", id+"@example.invalid")
	}
	exec(`INSERT INTO projects(id,code,name,address,project_type,start_date,end_date,manager_id,created_by) VALUES(?,?,?,'','building','2026-01-01','2026-12-31',?,?)`, f.Project, f.Project, "Test project", f.Creator, f.Creator)
	exec(`INSERT INTO sites(id,project_id,code,name,address,created_by) VALUES(?,?,'S','Site','',?)`, f.Site, f.Project, f.Creator)
	exec(`INSERT INTO site_logs(id,project_id,site_id,date,work_performed,weather,blockers,notes,status,created_by) VALUES(?,?,?,'2026-01-01','Work','','','','submitted',?)`, f.Log, f.Project, f.Site, f.Creator)
	return f
}
