package transactions

import (
	"buildestate/backend/internal/contracts"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type reviewTarget struct{ table, column, role, pending string }

func target(kind string) (reviewTarget, error) {
	switch kind {
	case "site-log":
		return reviewTarget{"site_logs", "site_log_id", "project-manager", "submitted"}, nil
	case "purchase-order":
		return reviewTarget{"purchase_orders", "purchase_order_id", "", "submitted"}, nil
	case "work-order":
		return reviewTarget{"work_orders", "work_order_id", "finance-officer", "submitted"}, nil
	case "expense":
		return reviewTarget{"expenses", "expense_id", "finance-officer", "submitted"}, nil
	case "reservation":
		return reviewTarget{"reservations", "reservation_id", "administrator", "pending-review"}, nil
	case "cancellation":
		return reviewTarget{"cancellations", "", "administrator", "submitted"}, nil
	}
	return reviewTarget{}, Invalid()
}

// ReviewPredicate is a parameterized eligibility predicate for a domain query
// whose subject table has alias r. Apply BEFORE ordering/limiting a review queue.
// Both queues and decisions read current account/assignment state; role values
// are never accepted from a request. Purchase routing uses its persisted snapshot.
func ReviewPredicate(kind, actor string) (string, []any, error) {
	t, err := target(kind)
	if err != nil {
		return "", nil, err
	}
	roleSQL := "u.role_code=?"
	args := []any{t.pending, actor}
	if kind == "purchase-order" {
		roleSQL = "((r.submitter_role='procurement-officer' AND u.role_code='project-manager') OR (r.submitter_role='project-manager' AND u.role_code='finance-officer'))"
	} else {
		args = append(args, t.role)
	}
	q := `r.status=? AND r.created_by<>? AND EXISTS (SELECT 1 FROM users u WHERE u.id=? AND u.status='active' AND u.setup_required=0 AND (u.role_code='administrator' OR (` + roleSQL + ` AND EXISTS(SELECT 1 FROM user_projects up WHERE up.user_id=u.id AND up.project_id=r.project_id))))`
	// actor in the account lookup precedes the role placeholder.
	args = append(args[:2], append([]any{actor}, args[2:]...)...)
	return q, args, nil
}

type Review struct {
	Kind, ID, ActorID, RequestID, Decision, Reason string
	Version                                        int64
	At                                             time.Time
}

// Decide runs within Service.Mutate's transaction. Workflow effect validates and
// writes the lifecycle/effects, using the supplied authoritative project/state.
// It must not commit. The final state/version is checked before evidence is added.
// Return the event to Service.Mutate, which appends it atomically with idempotency.
func Decide(ctx context.Context, tx contracts.Transaction, r Review, effect func(context.Context, contracts.Transaction, string, string, string) error) (contracts.CommittedResult, contracts.AuditEvent, error) {
	var result contracts.CommittedResult
	var event contracts.AuditEvent
	t, err := target(r.Kind)
	if err != nil {
		return result, event, err
	}
	if effect == nil || r.At.IsZero() || r.RequestID == "" {
		return result, event, Invalid()
	}
	var project, creator, status string
	var version int64
	err = tx.QueryRowContext(ctx, "SELECT project_id,created_by,status,version FROM "+t.table+" WHERE id=?", r.ID).Scan(&project, &creator, &status, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return result, event, &Error{404, "NOT_FOUND", "The record was not found."}
	}
	if err != nil {
		return result, event, err
	}
	if err = CheckVersion(version, r.Version); err != nil {
		return result, event, err
	}
	if creator == r.ActorID {
		return result, event, Conflict("INDEPENDENT_REVIEW_REQUIRED")
	}
	if status != t.pending {
		return result, event, Conflict("INVALID_TRANSITION")
	}
	predicate, args, _ := ReviewPredicate(r.Kind, r.ActorID)
	args = append([]any{r.ID}, args...)
	var eligible int
	if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM "+t.table+" r WHERE r.id=? AND "+predicate+")", args...).Scan(&eligible); err != nil {
		return result, event, err
	}
	if eligible != 1 {
		return result, event, &Error{403, "FORBIDDEN", "This review is not permitted."}
	}
	next := ""
	switch r.Decision {
	case "approve":
		next = "approved"
		if r.Kind == "reservation" {
			next = "cancelled"
		}
	case "return":
		if r.Kind == "site-log" || r.Kind == "purchase-order" || r.Kind == "work-order" {
			next = "returned"
		}
	case "reject":
		if r.Kind == "expense" || r.Kind == "cancellation" {
			next = "rejected"
		}
		if r.Kind == "reservation" {
			next = "active"
		}
	}
	if next == "" {
		return result, event, Conflict("INVALID_TRANSITION")
	}
	if (r.Decision != "approve" || r.Kind == "reservation") && strings.TrimSpace(r.Reason) == "" {
		return result, event, Invalid()
	}
	if err = effect(ctx, tx, project, status, next); err != nil {
		return result, event, err
	}
	var finalStatus string
	var finalVersion int64
	if err = tx.QueryRowContext(ctx, "SELECT status,version FROM "+t.table+" WHERE id=?", r.ID).Scan(&finalStatus, &finalVersion); err != nil {
		return result, event, err
	}
	if finalStatus != next || finalVersion != version+1 {
		return result, event, Conflict("INVALID_TRANSITION")
	}
	if r.Kind == "cancellation" {
		_, err = tx.ExecContext(ctx, `UPDATE cancellations SET decided_by=?,decided_at=?,decision_reason=? WHERE id=?`, r.ActorID, Timestamp(r.At), nullable(r.Reason), r.ID)
	} else {
		_, err = tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO review_decisions(id,project_id,%s,decision,subject_version,submitted_by,reviewer_id,reason,occurred_at) VALUES(?,?,?,?,?,?,?,?,?)`, t.column), NewID(), project, r.ID, r.Decision, version, creator, r.ActorID, nullable(r.Reason), Timestamp(r.At))
	}
	if err != nil {
		return result, event, err
	}
	result = contracts.CommittedResult{Status: 200, ResourceType: r.Kind, ResourceID: r.ID, Version: finalVersion}
	event = contracts.AuditEvent{ActorID: r.ActorID, RequestID: r.RequestID, Action: r.Kind + "." + r.Decision, ProjectID: &project, RecordType: r.Kind, RecordID: r.ID, OccurredAt: r.At, Changes: []contracts.SafeChange{{Field: "status", Before: &status, After: &next}}}
	if r.Reason != "" {
		event.Reason = &r.Reason
	}
	return result, event, nil
}
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
