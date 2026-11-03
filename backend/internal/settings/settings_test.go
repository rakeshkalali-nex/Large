package settings

import (
	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/posting"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
	"buildestate/backend/internal/testsupport"
	"context"
	"testing"
	"time"
)

type settingsClock struct{}

func (settingsClock) Now() time.Time { return time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC) }
func settingsFixture(t *testing.T) (Service, context.Context) {
	t.Helper()
	db := testsupport.Database(t)
	now := settingsClock{}.Now()
	for _, q := range []string{`INSERT INTO users(id,display_name,email,normalized_email,role_code,status,setup_required) VALUES('admin','Admin','admin@example.test','admin@example.test','administrator','active',0)`, `INSERT INTO sessions(id,token_digest,user_id,access_revision,credential_revision,created_at,last_interaction_at,idle_expires_at,absolute_expires_at) VALUES('session','digest','admin',1,1,'2026-11-03T10:00:00Z','2026-11-03T10:00:00Z','2026-11-03T11:00:00Z','2026-11-04T10:00:00Z')`} {
		if _, e := db.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	_ = now
	ctx := transport.WithRequestID(policy.WithPrincipal(context.Background(), policy.Principal{UserID: "admin", SessionDigest: "digest"}), "settings-test")
	return Service{D: contracts.Dependencies{DB: db, Clock: settingsClock{}}}, ctx
}
func validSettings() Settings {
	code := "USD"
	digits := 2
	notice := "notice-1"
	schedule := "schedule-1"
	return Settings{CompanyName: "Example", Branding: Branding{Wordmark: "Example"}, CurrencyCode: &code, CurrencyMinorDigits: &digits, Timezone: "UTC", TimezoneConfirmed: true, RecoveryInstructions: "Contact the owner", RecoveryContactLabel: "Owner", ReservationHours: 48, ReferenceValues: []ReferenceValue{{Key: "unit", Value: "each", Label: "Each"}}, PropertyTypes: []string{"House"}, InspectionChecklists: []Checklist{{Name: "Safety", Items: []string{"Check access"}}}, OperatingPolicy: OperatingPolicy{Countries: []string{"US"}, NoticeVersion: &notice, RetentionScheduleVersion: &schedule}, ConfirmOperationalPolicy: true}
}
func TestSettingsPersistentHistoryPostingAndLock(t *testing.T) {
	s, ctx := settingsFixture(t)
	v, e := s.saveSettings(ctx, validSettings())
	if e != nil {
		t.Fatal(e)
	}
	got, e := readSettings(ctx, s.D.DB)
	if e != nil || got.Version != 1 || len(got.PropertyTypes) != 1 || len(got.InspectionChecklists) != 1 {
		t.Fatalf("read: %+v %v", got, e)
	}
	e = (transactions.Manager{DB: s.D.DB}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		return (posting.Configuration{}).LockCurrency(ctx, tx)
	})
	if e != nil {
		t.Fatal(e)
	}
	other := "EUR"
	v.CurrencyCode = &other
	if _, e = s.saveSettings(ctx, v); e == nil {
		t.Fatal("changed locked currency")
	}
	var count int
	s.D.DB.QueryRow(`SELECT count(*) FROM settings_history`).Scan(&count)
	if count != 1 {
		t.Fatal("history changed after failure")
	}
	if _, e = s.saveSettings(ctx, validSettings()); e == nil {
		t.Fatal("accepted stale version")
	}
}
func TestSettingsValidationAndAuditRollback(t *testing.T) {
	s, ctx := settingsFixture(t)
	for _, mutate := range []func(*Settings){func(v *Settings) { v.Timezone = "Local" }, func(v *Settings) { x := "ZZZ"; v.CurrencyCode = &x }, func(v *Settings) { v.OperatingPolicy.Countries = nil }, func(v *Settings) { v.ReservationHours = 0 }, func(v *Settings) { v.PropertyTypes = []string{"House", "House"} }} {
		v := validSettings()
		mutate(&v)
		if _, e := s.saveSettings(ctx, v); e == nil {
			t.Fatal("accepted invalid setup")
		}
	}
	if _, e := s.D.DB.Exec(`CREATE TRIGGER fail_settings_audit BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT,'test fault'); END`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.saveSettings(ctx, validSettings()); e == nil {
		t.Fatal("ignored audit failure")
	}
	var n int
	s.D.DB.QueryRow(`SELECT count(*) FROM company_settings`).Scan(&n)
	if n != 0 {
		t.Fatal("partial settings commit")
	}
}
func TestSettingsPolicyGateAndAuthorization(t *testing.T) {
	s, ctx := settingsFixture(t)
	v := validSettings()
	v.ConfirmOperationalPolicy = false
	v.TimezoneConfirmed = false
	if _, e := s.saveSettings(ctx, v); e != nil {
		t.Fatal(e)
	}
	e := (transactions.Manager{DB: s.D.DB}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		_, e := (posting.Configuration{}).RequireReady(ctx, tx)
		return e
	})
	if e == nil {
		t.Fatal("unconfirmed setup allowed posting")
	}
	if _, e = s.saveSettings(context.Background(), validSettings()); e == nil {
		t.Fatal("anonymous mutation")
	}
}

func TestPolicyConfirmationSurvivesBrandingButNotRuleChanges(t *testing.T) {
	s, ctx := settingsFixture(t)
	v, e := s.saveSettings(ctx, validSettings())
	if e != nil {
		t.Fatal(e)
	}
	confirmed := *v.OperatingPolicy.ConfirmedAt
	v.CompanyName = "Renamed"
	v.ConfirmOperationalPolicy = false
	v, e = s.saveSettings(ctx, v)
	if e != nil {
		t.Fatal(e)
	}
	if v.OperatingPolicy.ConfirmedAt == nil || *v.OperatingPolicy.ConfirmedAt != confirmed {
		t.Fatal("branding revoked policy confirmation")
	}
	v.ReferenceValues = append(v.ReferenceValues, ReferenceValue{Key: "privacy.consent-required", Value: "false", Label: "Owner approved rule"})
	v, e = s.saveSettings(ctx, v)
	if e != nil {
		t.Fatal(e)
	}
	if v.OperatingPolicy.ConfirmedAt != nil {
		t.Fatal("changed policy remained confirmed")
	}
}

func TestPersonalDataGateRequiresLocalPolicy(t *testing.T) {
	s, ctx := settingsFixture(t)
	if e := RequirePersonalDataReady(ctx, s.D.DB); e == nil {
		t.Fatal("missing policy permitted")
	}
	v := validSettings()
	v.CurrencyCode = nil
	v.CurrencyMinorDigits = nil
	if _, e := s.saveSettings(ctx, v); e != nil {
		t.Fatal(e)
	}
	if e := RequirePersonalDataReady(ctx, s.D.DB); e != nil {
		t.Fatal(e)
	}
	if _, e := s.D.DB.Exec(`UPDATE company_settings SET operational_policy_confirmed_at=NULL`); e != nil {
		t.Fatal(e)
	}
	if e := RequirePersonalDataReady(ctx, s.D.DB); e == nil {
		t.Fatal("unconfirmed policy permitted")
	}
}
