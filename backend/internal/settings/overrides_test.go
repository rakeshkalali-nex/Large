package settings

import (
	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/platform/transport"
	"context"
	"testing"
)

func TestOverrideQuantityScaleAndSafeguardReferences(t *testing.T) {
	s, ctx := settingsFixture(t)
	overrideSeed(t, s)
	in := ProjectOverrides{ProjectID: "p", StockMinimums: []StockMinimum{{MaterialID: "m", MinimumQuantity: "1.250"}}}
	if err := s.saveProjectOverrides(ctx, &in); err != nil {
		t.Fatal(err)
	}
	var stored int64
	if err := s.D.DB.QueryRow(`SELECT minimum_quantity FROM material_site_minimums WHERE project_id='p'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	got, err := readProjectOverrides(ctx, s.D.DB, "p")
	if err != nil || stored != 1250 || got.StockMinimums[0].MinimumQuantity != "1.25" {
		t.Fatalf("scale: %d %+v %v", stored, got, err)
	}
	for _, key := range []string{"privacy.consent-required", "privacy.consentRequired", "authorization", "approval.independent"} {
		got.ReferenceValues = []ReferenceValue{{Key: key, Value: "false"}}
		if err := s.saveProjectOverrides(ctx, &got); err == nil {
			t.Fatalf("safeguard override %s", key)
		}
	}
	got.ReferenceValues = nil
	got.StockMinimums[0].MinimumQuantity = "1.2500"
	if err := s.saveProjectOverrides(ctx, &got); err == nil {
		t.Fatal("excess precision accepted")
	}
}

func overrideSeed(t *testing.T, s Service) {
	t.Helper()
	for _, q := range []string{
		`INSERT INTO projects(id,code,name,address,project_type,start_date,end_date,manager_id,created_by) VALUES('p','P','Project','','building','2026-01-01','2026-12-31','admin','admin')`,
		`INSERT INTO sites(id,project_id,code,name,address,created_by) VALUES('site','p','S','Site','','admin')`,
		`INSERT INTO materials(id,code,name,unit,minimum_quantity,created_by) VALUES('m','M','Material','each',0,'admin')`,
		`INSERT INTO inspection_checklists(id,name,revision,status,created_by) VALUES('check','Safety',1,'active','admin')`,
	} {
		if _, err := s.D.DB.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
}
func overrideManager(t *testing.T, s Service) context.Context {
	t.Helper()
	for _, q := range []string{
		`INSERT INTO users(id,display_name,email,normalized_email,role_code,status,setup_required) VALUES('pm','Manager','pm@example.test','pm@example.test','project-manager','active',0)`,
		`INSERT INTO sessions(id,token_digest,user_id,access_revision,credential_revision,created_at,last_interaction_at,idle_expires_at,absolute_expires_at) VALUES('pm-session','pm-digest','pm',1,1,'2026-11-03T10:00:00Z','2026-11-03T10:00:00Z','2026-11-03T11:00:00Z','2026-11-04T10:00:00Z')`,
		`INSERT INTO user_projects(user_id,project_id,created_by) VALUES('pm','p','admin')`,
	} {
		if _, err := s.D.DB.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return transport.WithRequestID(policy.WithPrincipal(context.Background(), policy.Principal{UserID: "pm", SessionDigest: "pm-digest"}), "override-test")
}
func TestOverridesPersistScopeAndSeparateMinimumPermission(t *testing.T) {
	s, ctx := settingsFixture(t)
	overrideSeed(t, s)
	hours := 72
	site := "site"
	in := ProjectOverrides{ProjectID: "p", ReservationHours: &hours, StockMinimums: []StockMinimum{{SiteID: &site, MaterialID: "m", MinimumQuantity: "12"}}, InspectionChecklistIDs: []string{"check"}, ReferenceValues: []ReferenceValue{{Key: "weather", Value: "dry"}}}
	if err := s.saveProjectOverrides(ctx, &in); err != nil {
		t.Fatal(err)
	}
	got, err := readProjectOverrides(ctx, s.D.DB, "p")
	if err != nil || got.Version != 1 || got.ReservationHours == nil || *got.ReservationHours != 72 || len(got.StockMinimums) != 1 || len(got.ReferenceValues) != 1 || len(got.InspectionChecklistIDs) != 1 {
		t.Fatalf("persisted: %+v %v", got, err)
	}
	pm := overrideManager(t, s)
	if err = s.saveProjectOverrides(pm, &got); err != nil {
		t.Fatalf("manager may retain existing minimums: %v", err)
	}
	got.StockMinimums[0].MinimumQuantity = "13"
	if err = s.saveProjectOverrides(pm, &got); err == nil {
		t.Fatal("manager changed stock minimum")
	}
	got.StockMinimums[0].MinimumQuantity = "12"
	got.ProjectID = "unassigned"
	if err = s.saveProjectOverrides(pm, &got); err == nil {
		t.Fatal("unassigned project allowed")
	}
	if err = s.saveProjectOverrides(ctx, &in); err == nil {
		t.Fatal("stale edit allowed")
	}
}
func TestOverridesValidationAndAuditRollback(t *testing.T) {
	s, ctx := settingsFixture(t)
	overrideSeed(t, s)
	for _, in := range []ProjectOverrides{
		{ProjectID: "p", StockMinimums: []StockMinimum{{MaterialID: "m", MinimumQuantity: "-1"}}},
		{ProjectID: "p", StockMinimums: []StockMinimum{{MaterialID: "missing", MinimumQuantity: "1"}}},
		{ProjectID: "p", InspectionChecklistIDs: []string{"missing"}},
		{ProjectID: "p", ReferenceValues: []ReferenceValue{{Key: "x", Value: "y"}, {Key: "x", Value: "y"}}},
	} {
		if err := s.saveProjectOverrides(ctx, &in); err == nil {
			t.Fatalf("accepted %+v", in)
		}
	}
	if _, err := s.D.DB.Exec(`CREATE TRIGGER fail_overrides_audit BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT,'fault'); END`); err != nil {
		t.Fatal(err)
	}
	in := ProjectOverrides{ProjectID: "p", InspectionChecklistIDs: []string{"check"}}
	if err := s.saveProjectOverrides(ctx, &in); err == nil {
		t.Fatal("audit failure ignored")
	}
	got, err := readProjectOverrides(ctx, s.D.DB, "p")
	if err != nil || got.Version != 0 || len(got.InspectionChecklistIDs) != 0 {
		t.Fatalf("partial commit: %+v %v", got, err)
	}
}
