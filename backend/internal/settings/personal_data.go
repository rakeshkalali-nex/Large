package settings

import (
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// RequirePersonalDataReady must be called by personal-record creation services
// inside their own write transaction. Bootstrap identity is a separate operation.
// This gate does not substitute for subject authorization or notice recording.
func RequirePersonalDataReady(ctx context.Context, tx contracts.Transaction) error {
	var notice, schedule, confirmed sql.NullString
	if tx == nil {
		return transactions.Conflict("OPERATING_POLICY_REQUIRED")
	}
	e := tx.QueryRowContext(ctx, `SELECT notice_version,retention_schedule_version,operational_policy_confirmed_at FROM company_settings WHERE id='company'`).Scan(&notice, &schedule, &confirmed)
	if errors.Is(e, sql.ErrNoRows) {
		return transactions.Conflict("OPERATING_POLICY_REQUIRED")
	}
	if e != nil {
		return e
	}
	if strings.TrimSpace(notice.String) == "" || strings.TrimSpace(schedule.String) == "" || !confirmed.Valid {
		return transactions.Conflict("OPERATING_POLICY_REQUIRED")
	}
	if _, e = time.Parse(time.RFC3339Nano, confirmed.String); e != nil {
		return transactions.Conflict("OPERATING_POLICY_REQUIRED")
	}
	var countries int
	e = tx.QueryRowContext(ctx, `SELECT count(*) FROM operating_countries WHERE settings_id='company' AND country_code GLOB '[A-Z][A-Z]'`).Scan(&countries)
	if e != nil {
		return e
	}
	if countries == 0 {
		return transactions.Conflict("OPERATING_POLICY_REQUIRED")
	}
	return nil
}
