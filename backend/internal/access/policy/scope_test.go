package policy

import (
	"testing"
)

func TestAllAllowedQueryPlansExecute(t *testing.T) {
	for role := range roleGrants {
		t.Run(role, func(t *testing.T) {
			s, ctx, _ := fixture(t, role)
			for name, r := range resources {
				scope, err := s.Scope(ctx, name)
				if !hasCapability(role, r.capability) {
					wantCode(t, err, 403, "FORBIDDEN")
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				rows, err := s.DB.Query("SELECT "+scope.Columns+" FROM "+scope.Table+" WHERE "+scope.Predicate+" LIMIT 1", scope.Args...)
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				rows.Close()
			}
			wantCode(t, func() error { _, err := s.Scope(ctx, "unknown"); return err }(), 403, "FORBIDDEN")
		})
	}
}

func TestTaskAssignmentIsReadFromRecord(t *testing.T) {
	s, ctx, _ := fixture(t, "site-engineer")
	exec(t, s.DB, `INSERT INTO users(id,display_name,email,normalized_email,role_code) VALUES('v','Other','v@example.test','v@example.test','site-engineer')`)
	for _, row := range []struct{ id, project, owner string }{{"own", "p", "u"}, {"other-owner", "p", "v"}, {"other-project", "other", "u"}} {
		exec(t, s.DB, `INSERT INTO tasks(id,project_id,name,responsible_id,start_date,end_date,created_by) VALUES(?,?,'Task',?,'2026-01-01','2026-12-31','u')`, row.id, row.project, row.owner)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = s.RequireAssignedTaskTx(ctx, tx, "own"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"other-owner", "other-project", "missing"} {
		wantCode(t, s.RequireAssignedTaskTx(ctx, tx, id), 404, "NOT_FOUND")
	}
	if _, err = tx.Exec(`UPDATE tasks SET responsible_id='v' WHERE id='own'`); err != nil {
		t.Fatal(err)
	}
	wantCode(t, s.RequireAssignedTaskTx(ctx, tx, "own"), 404, "NOT_FOUND")
}

func TestNoProjectsNeverFallsBackToCompanyScope(t *testing.T) {
	s, ctx, _ := fixture(t, "project-manager")
	exec(t, s.DB, `DELETE FROM user_projects WHERE user_id='u' AND project_id='p'`)
	if err := s.Authorize(ctx, "projects.read", ""); err != nil {
		t.Fatal("collection shell should remain available", err)
	}
	scope, err := s.Scope(ctx, "projects")
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err = s.DB.QueryRow("SELECT count(*) FROM "+scope.Table+" WHERE ("+scope.Predicate+") AND r.id IN ('p','other')", scope.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("unassigned user sees projects")
	}
}
