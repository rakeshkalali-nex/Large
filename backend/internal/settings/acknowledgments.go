package settings

import (
	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
)

type PolicyAcknowledgment struct {
	ID               string `json:"id"`
	SubjectType      string `json:"subjectType"`
	SubjectID        string `json:"subjectId"`
	NoticeVersion    string `json:"noticeVersion"`
	AcknowledgedAt   string `json:"acknowledgedAt"`
	ConsentRequired  bool   `json:"consentRequired"`
	ConsentGiven     *bool  `json:"consentGiven"`
	RecordedByReason string `json:"recordedByReason"`
	CreatedAt        string `json:"createdAt"`
}

func (s Service) registerAcknowledgments(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/policy-acknowledgments", s.handle("privacy.acknowledge", func(w http.ResponseWriter, r *http.Request) {
		var in PolicyAcknowledgment
		err := decodeOwnedSettings(w, r, &in)
		if err == nil {
			err = s.saveAcknowledgment(r.Context(), &in)
		}
		respond(w, r, in, err)
	}))
}
func acknowledgmentDenied() error {
	return &transactions.Error{Status: 403, Code: "FORBIDDEN", Message: "This operation is not permitted."}
}
func (s Service) saveAcknowledgment(ctx context.Context, in *PolicyAcknowledgment) error {
	at, err := time.Parse(time.RFC3339Nano, in.AcknowledgedAt)
	if err != nil || s.D.Clock == nil || at.After(s.now()) || in.SubjectID == "" || strings.TrimSpace(in.NoticeVersion) == "" || strings.TrimSpace(in.RecordedByReason) == "" || len(in.RecordedByReason) > 2000 {
		return invalid()
	}
	in.ID = transactions.NewID()
	in.CreatedAt = s.now().UTC().Format(time.RFC3339Nano)
	err = s.write(ctx, "privacy.acknowledge", "", "settings.confirm", "settings", in.ID, func(ctx context.Context, tx contracts.Transaction) error {
		if err := RequirePersonalDataReady(ctx, tx); err != nil {
			return err
		}
		var notice, schedule, confirmed sql.NullString
		var countries int
		err := tx.QueryRowContext(ctx, `SELECT notice_version,retention_schedule_version,operational_policy_confirmed_at,(SELECT count(*) FROM operating_countries WHERE settings_id='company') FROM company_settings WHERE id='company'`).Scan(&notice, &schedule, &confirmed, &countries)
		if errors.Is(err, sql.ErrNoRows) {
			return invalid()
		}
		if err != nil {
			return err
		}
		if !notice.Valid || notice.String != in.NoticeVersion || !schedule.Valid || strings.TrimSpace(schedule.String) == "" || !confirmed.Valid || confirmed.String == "" || countries == 0 {
			return invalid()
		}
		actor := policy.PrincipalFrom(ctx).UserID
		// The confirmed company policy, never a client flag or project override,
		// determines whether explicit consent is required.
		var consentPolicy string
		if err = tx.QueryRowContext(ctx, `SELECT coalesce((SELECT value FROM reference_values WHERE project_id IS NULL AND key='privacy.consent-required' AND status='active' ORDER BY value DESC LIMIT 1),'true')`).Scan(&consentPolicy); err != nil {
			return err
		}
		required := consentPolicy != "false"
		if in.ConsentRequired != required {
			return invalid()
		}
		if in.ConsentRequired && (in.ConsentGiven == nil || !*in.ConsentGiven) {
			return invalid()
		}
		var role string
		if err = tx.QueryRowContext(ctx, `SELECT role_code FROM users WHERE id=?`, actor).Scan(&role); err != nil {
			return err
		}
		var owner, project string
		switch in.SubjectType {
		case "user":
			err = tx.QueryRowContext(ctx, `SELECT id,'' FROM users WHERE id=? AND status='active'`, in.SubjectID).Scan(&owner, &project)
		case "lead":
			err = tx.QueryRowContext(ctx, `SELECT sales_executive_id,project_id FROM leads WHERE id=?`, in.SubjectID).Scan(&owner, &project)
		case "customer":
			err = tx.QueryRowContext(ctx, `SELECT coalesce(l.sales_executive_id,c.created_by),c.project_id FROM customers c LEFT JOIN leads l ON l.id=c.source_lead_id AND l.project_id=c.project_id WHERE c.id=? AND c.status='active'`, in.SubjectID).Scan(&owner, &project)
		case "supplier-contact":
			err = tx.QueryRowContext(ctx, `SELECT s.created_by,'' FROM supplier_contacts c JOIN suppliers s ON s.id=c.supplier_id WHERE c.id=? AND s.status='active'`, in.SubjectID).Scan(&owner, &project)
		case "contractor-contact":
			err = tx.QueryRowContext(ctx, `SELECT s.created_by,'' FROM contractor_contacts c JOIN contractors s ON s.id=c.contractor_id WHERE c.id=? AND s.status='active'`, in.SubjectID).Scan(&owner, &project)
		default:
			return invalid()
		}
		if errors.Is(err, sql.ErrNoRows) {
			return policy.NotFound()
		}
		if err != nil {
			return err
		}
		if role != "administrator" && owner != actor {
			return acknowledgmentDenied()
		}
		if project != "" {
			if err = (policy.Service{Clock: s.D.Clock}).AuthorizeTx(ctx, tx, "privacy.acknowledge", project); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO policy_acknowledgments(id,subject_type,subject_id,notice_version,acknowledged_at,consent_required,consent_given,recorded_by_reason,created_by,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, in.ID, in.SubjectType, in.SubjectID, in.NoticeVersion, at.UTC().Format(time.RFC3339Nano), in.ConsentRequired, in.ConsentGiven, in.RecordedByReason, actor, in.CreatedAt)
		if err != nil {
			return err
		}
		var consentID *string
		if in.ConsentGiven != nil && *in.ConsentGiven {
			consentID = &in.ID
		}
		switch in.SubjectType {
		case "lead":
			_, err = tx.ExecContext(ctx, `UPDATE leads SET notice_acknowledgment_id=?,consent_id=?,updated_at=?,version=version+1 WHERE id=?`, in.ID, consentID, in.CreatedAt, in.SubjectID)
		case "customer":
			_, err = tx.ExecContext(ctx, `UPDATE customers SET notice_acknowledgment_id=?,consent_id=?,updated_at=?,version=version+1 WHERE id=?`, in.ID, consentID, in.CreatedAt, in.SubjectID)
		}
		return err
	})
	if err == nil {
		if replay, ok := mutationReplay(ctx).(PolicyAcknowledgment); ok {
			*in = replay
		}
	}
	return err
}
