package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
)

type mutationContextKey struct{}

// Mutation tracks a trusted request's durable result; never accept actor grants
// from the body. Direct service callers can attach an explicit mutation as well.
type Mutation struct {
	Key, Digest, Operation string
	Replay                 any
	Result                 *contracts.CommittedResult
}

func WithMutation(ctx context.Context, m *Mutation) context.Context {
	return context.WithValue(ctx, mutationContextKey{}, m)
}
func mutationFrom(ctx context.Context) *Mutation {
	m, _ := ctx.Value(mutationContextKey{}).(*Mutation)
	return m
}
func mutationRequest(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if e != nil {
		return r, invalid()
	}
	var body any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&body) != nil {
		return r, invalid()
	}
	if decoder.Decode(new(any)) != io.EOF {
		return r, invalid()
	}
	digest, e := transactions.Digest(body)
	if e != nil {
		return r, e
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		return r, invalid()
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	return r.WithContext(WithMutation(r.Context(), &Mutation{Key: key, Digest: digest, Operation: r.Method + " " + r.URL.Path})), nil
}
func mutationIdentity(ctx context.Context, actor string, m *Mutation) contracts.MutationIdentity {
	return contracts.MutationIdentity{ActorID: actor, RequestID: requestID(ctx), Operation: m.Operation, IdempotencyKey: m.Key, RequestDigest: m.Digest}
}
func mutationProjection(ctx context.Context, tx contracts.Transaction, result contracts.CommittedResult) (any, error) {
	switch result.ResourceType {
	case "settings":
		var raw string
		if e := tx.QueryRowContext(ctx, `SELECT snapshot_json FROM settings_history WHERE version=?`, result.Version).Scan(&raw); e != nil {
			return nil, e
		}
		var v Settings
		e := json.Unmarshal([]byte(raw), &v)
		return v, e
	case "project":
		return readProjectOverrides(ctx, tx, result.ResourceID)
	case "policy-acknowledgment":
		var v PolicyAcknowledgment
		e := tx.QueryRowContext(ctx, `SELECT id,subject_type,subject_id,notice_version,acknowledged_at,consent_required,consent_given,recorded_by_reason,created_at FROM policy_acknowledgments WHERE id=?`, result.ResourceID).Scan(&v.ID, &v.SubjectType, &v.SubjectID, &v.NoticeVersion, &v.AcknowledgedAt, &v.ConsentRequired, &v.ConsentGiven, &v.RecordedByReason, &v.CreatedAt)
		if e != nil {
			return nil, e
		}
		actor := policy.PrincipalFrom(ctx).UserID
		var role string
		if e = tx.QueryRowContext(ctx, `SELECT role_code FROM users WHERE id=?`, actor).Scan(&role); e != nil {
			return nil, e
		}
		if role != "administrator" {
			owner, project := "", ""
			switch v.SubjectType {
			case "user":
				owner = v.SubjectID
			case "lead":
				e = tx.QueryRowContext(ctx, `SELECT sales_executive_id,project_id FROM leads WHERE id=?`, v.SubjectID).Scan(&owner, &project)
			case "customer":
				e = tx.QueryRowContext(ctx, `SELECT coalesce(l.sales_executive_id,c.created_by),c.project_id FROM customers c LEFT JOIN leads l ON l.id=c.source_lead_id WHERE c.id=?`, v.SubjectID).Scan(&owner, &project)
			case "supplier-contact":
				e = tx.QueryRowContext(ctx, `SELECT s.created_by FROM suppliers s JOIN supplier_contacts c ON c.supplier_id=s.id WHERE c.id=?`, v.SubjectID).Scan(&owner)
			case "contractor-contact":
				e = tx.QueryRowContext(ctx, `SELECT s.created_by FROM contractors s JOIN contractor_contacts c ON c.contractor_id=s.id WHERE c.id=?`, v.SubjectID).Scan(&owner)
			}
			if e != nil {
				return nil, e
			}
			if owner != actor {
				return nil, acknowledgmentDenied()
			}
			if project != "" {
				var assigned int
				if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM user_projects WHERE user_id=? AND project_id=?`, actor, project).Scan(&assigned); e != nil {
					return nil, e
				}
				if assigned == 0 {
					return nil, acknowledgmentDenied()
				}
			}
		}
		return v, nil
	default:
		resources := map[string]string{"privacy-request": "privacy-requests", "legal-hold": "retention-holds", "retention-review": "retention-reviews"}
		if resource := resources[result.ResourceType]; resource != "" {
			return stewardRead(ctx, tx, resource, result.ResourceID)
		}
	}
	return nil, invalid()
}
func mutationResult(ctx context.Context, tx contracts.Transaction, action, kind, id string) (contracts.CommittedResult, error) {
	result := contracts.CommittedResult{Status: 200, ResourceType: kind, ResourceID: id, Version: 1}
	if action == "settings.confirm" {
		result.ResourceType = "policy-acknowledgment"
		return result, nil
	}
	tables := map[string]string{"settings": "company_settings", "project": "project_overrides", "privacy-request": "privacy_requests", "legal-hold": "retention_holds", "retention-review": "retention_reviews"}
	table := tables[kind]
	if table == "" {
		return result, invalid()
	}
	column := "id"
	if kind == "project" {
		column = "project_id"
	}
	e := tx.QueryRowContext(ctx, "SELECT version FROM "+table+" WHERE "+column+"=?", id).Scan(&result.Version)
	return result, e
}
func mutationReplay(ctx context.Context) any {
	if m := mutationFrom(ctx); m != nil {
		return m.Replay
	}
	return nil
}
func isMutation(r *http.Request) bool { return r.Method == "POST" || r.Method == "PUT" }
