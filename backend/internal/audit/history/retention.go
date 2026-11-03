package history

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"buildestate/backend/internal/contracts"
)

// Retention is a read-only policy projection, not permission to delete evidence.
// EligibleAt is absent until the retention clock can start, or while held.
type Retention struct {
	Class      string   `json:"class"`
	EligibleAt *string  `json:"eligibleAt"`
	HoldIDs    []string `json:"holdIds"`
}

func retentionFor(ctx context.Context, tx contracts.Transaction, id string) (Retention, error) {
	result := Retention{HoldIDs: []string{}}
	var action, occurred, kind, record string
	var project, closed, status sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT a.retention_class,a.action,a.occurred_at,a.project_id,a.record_type,a.record_id,p.closed_at,p.status
		FROM audit_events a LEFT JOIN projects p ON p.id=COALESCE(a.project_id,CASE WHEN a.record_type='project' THEN a.record_id END)
		WHERE a.id=?`, id).Scan(&result.Class, &action, &occurred, &project, &kind, &record, &closed, &status)
	if err != nil {
		return result, err
	}
	// Legacy writers classified every auth.* action as authentication. Only this
	// exact allowlist is routine; password/setup and unknown actions retain evidence.
	routine := false
	switch action {
	case "auth.login.succeeded", "auth.login.failed", "auth.logout", "auth.session.expired", "auth.access.denied":
		routine = true
	}
	if !routine {
		result.Class = "business"
	}
	result.HoldIDs, err = retentionHolds(ctx, tx, id)
	if err != nil {
		return result, err
	}
	if len(result.HoldIDs) != 0 {
		return result, nil
	}
	base, years := occurred, 1
	if result.Class != "authentication" {
		if !closed.Valid || closed.String == "" || (status.String != "completed" && status.String != "archived") {
			return result, nil
		}
		base, years = closed.String, 7
	}
	stamp, err := time.Parse(time.RFC3339Nano, base)
	if err != nil {
		stamp, err = time.Parse("2006-01-02", base)
	}
	if err != nil {
		return result, errors.New("invalid audit retention timestamp")
	}
	eligible := stamp.UTC().AddDate(years, 0, 0).Format(time.RFC3339Nano)
	result.EligibleAt = &eligible
	return result, nil
}

// UNION (not UNION ALL) makes cyclic retained snapshot references terminate.
// Snapshot source identities use the existing contracts.RecordVersion resource/id
// shape. recordType/recordId is also accepted for retained record references.
// This follows persisted references only; it never invents report filter membership.
func retentionHolds(ctx context.Context, tx contracts.Transaction, id string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `WITH RECURSIVE sources(kind,id) AS (
		SELECT 'audit-event', ?
		UNION SELECT a.record_type,a.record_id FROM audit_events a JOIN sources s ON s.kind='audit-event' AND s.id=a.id
		UNION SELECT 'project',a.project_id FROM audit_events a JOIN sources s ON s.kind='audit-event' AND s.id=a.id WHERE a.project_id IS NOT NULL
		UNION SELECT 'report-snapshot',r.id FROM report_snapshots r JOIN sources s ON s.kind='document' AND s.id=r.document_id
		UNION SELECT 'document',r.document_id FROM report_snapshots r JOIN sources s ON s.kind='report-snapshot' AND s.id=r.id
		UNION SELECT 'project',r.project_id FROM report_snapshots r JOIN sources s ON s.kind='report-snapshot' AND s.id=r.id
		UNION SELECT COALESCE(json_extract(j.value,'$.resource'),json_extract(j.value,'$.recordType')),
			COALESCE(json_extract(j.value,'$.id'),json_extract(j.value,'$.recordId'))
			FROM report_snapshots r JOIN sources s ON s.kind='report-snapshot' AND s.id=r.id,
			json_each(r.source_versions_json) j WHERE j.type='object'
	) SELECT DISTINCT h.id FROM retention_holds h JOIN sources s ON s.kind=h.record_type AND s.id=h.record_id
		WHERE h.status='active' ORDER BY h.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var hold string
		if err := rows.Scan(&hold); err != nil {
			return nil, err
		}
		ids = append(ids, hold)
	}
	return ids, rows.Err()
}
