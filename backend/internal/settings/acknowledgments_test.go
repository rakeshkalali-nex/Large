package settings

import (
	"database/sql"
	"testing"
)

func TestAcknowledgmentOptionalConsentPolicy(t *testing.T) {
	s, ctx := settingsFixture(t)
	v := validSettings()
	v.ReferenceValues = append(v.ReferenceValues, ReferenceValue{Key: "privacy.consent-required", Value: "false", Label: "Consent policy"})
	if _, err := s.saveSettings(ctx, v); err != nil {
		t.Fatal(err)
	}
	for _, given := range []*bool{nil, new(bool)} {
		in := acknowledgmentInput()
		in.ConsentRequired = false
		in.ConsentGiven = given
		if err := s.saveAcknowledgment(ctx, &in); err != nil {
			t.Fatal(err)
		}
		var required bool
		var stored sql.NullBool
		if err := s.D.DB.QueryRow(`SELECT consent_required,consent_given FROM policy_acknowledgments WHERE id=?`, in.ID).Scan(&required, &stored); err != nil {
			t.Fatal(err)
		}
		if required || stored.Valid != (given != nil) || stored.Bool {
			t.Fatal("optional consent not preserved")
		}
	}
	in := acknowledgmentInput()
	if err := s.saveAcknowledgment(ctx, &in); err == nil {
		t.Fatal("mismatched client policy accepted")
	}
}

func acknowledgmentInput() PolicyAcknowledgment {
	yes := true
	return PolicyAcknowledgment{SubjectType: "user", SubjectID: "admin", NoticeVersion: "notice-1", AcknowledgedAt: "2026-11-03T09:00:00Z", ConsentRequired: true, ConsentGiven: &yes, RecordedByReason: "Notice explained in person"}
}
func TestAcknowledgmentPolicyConsentAndExistence(t *testing.T) {
	s, ctx := settingsFixture(t)
	in := acknowledgmentInput()
	if err := s.saveAcknowledgment(ctx, &in); err == nil {
		t.Fatal("missing policy allowed")
	}
	if _, err := s.saveSettings(ctx, validSettings()); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*PolicyAcknowledgment){
		func(v *PolicyAcknowledgment) { v.ConsentRequired = false },
		func(v *PolicyAcknowledgment) { v.ConsentGiven = nil },
		func(v *PolicyAcknowledgment) { no := false; v.ConsentGiven = &no },
		func(v *PolicyAcknowledgment) { v.NoticeVersion = "obsolete" },
		func(v *PolicyAcknowledgment) { v.SubjectID = "missing" },
		func(v *PolicyAcknowledgment) { v.AcknowledgedAt = "2027-01-01T00:00:00Z" },
	} {
		v := acknowledgmentInput()
		change(&v)
		if err := s.saveAcknowledgment(ctx, &v); err == nil {
			t.Fatalf("accepted %+v", v)
		}
	}
	if err := s.saveAcknowledgment(ctx, &in); err != nil {
		t.Fatal(err)
	}
	var required, given int
	if err := s.D.DB.QueryRow(`SELECT consent_required,consent_given FROM policy_acknowledgments WHERE id=?`, in.ID).Scan(&required, &given); err != nil {
		t.Fatal(err)
	}
	if required != 1 || given != 1 {
		t.Fatal("client weakened consent")
	}
}
func TestAcknowledgmentOwnerAndAuditAtomicity(t *testing.T) {
	s, ctx := settingsFixture(t)
	overrideSeed(t, s)
	pm := overrideManager(t, s)
	if _, err := s.saveSettings(ctx, validSettings()); err != nil {
		t.Fatal(err)
	}
	in := acknowledgmentInput()
	if err := s.saveAcknowledgment(pm, &in); err == nil {
		t.Fatal("non-owner acknowledged another user")
	}
	in.SubjectID = "pm"
	if err := s.saveAcknowledgment(pm, &in); err != nil {
		t.Fatal(err)
	}
	if _, err := s.D.DB.Exec(`CREATE TRIGGER fail_ack_audit BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT,'fault'); END`); err != nil {
		t.Fatal(err)
	}
	in = acknowledgmentInput()
	if err := s.saveAcknowledgment(ctx, &in); err == nil {
		t.Fatal("audit failure ignored")
	}
	var n int
	if err := s.D.DB.QueryRow(`SELECT count(*) FROM policy_acknowledgments`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("partial acknowledgment committed")
	}
}
