package settings

import (
	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/audit/writer"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
	"context"
	"net/http"
	"time"
)

type Service struct{ D contracts.Dependencies }

func invalid() error                       { return transactions.Invalid() }
func requestID(ctx context.Context) string { return transport.RequestID(ctx) }
func (s Service) now() time.Time           { return s.D.Clock.Now() }
func (s Service) actor(ctx context.Context, tx contracts.Transaction) (string, error) {
	p := policy.PrincipalFrom(ctx)
	if p.UserID == "" {
		return "", invalid()
	}
	return p.UserID, nil
}
func (s Service) write(ctx context.Context, capability, projectID, action, recordType, recordID string, fn func(context.Context, contracts.Transaction) error) error {
	if m := mutationFrom(ctx); m != nil {
		m.Replay = nil
		m.Result = nil
	}
	if s.D.DB == nil || s.D.Clock == nil {
		return &transactions.Error{Status: 503, Code: "SERVICE_UNAVAILABLE", Message: "Service unavailable."}
	}
	manager := s.D.Transactions
	if manager == nil {
		manager = transactions.Manager{DB: s.D.DB}
	}
	audit := s.D.Audit
	if audit == nil {
		audit = writer.New()
	}
	return manager.WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		if err := (policy.Service{Clock: s.D.Clock}).AuthorizeTx(ctx, tx, capability, projectID); err != nil {
			return err
		}
		actor, err := s.actor(ctx, tx)
		if err != nil {
			return err
		}
		m := mutationFrom(ctx)
		store := transactions.Store{Now: s.now}
		if m != nil {
			prior, e := store.Lookup(ctx, tx, mutationIdentity(ctx, actor, m))
			if e != nil {
				return e
			}
			if prior != nil {
				m.Result = prior
				m.Replay, e = mutationProjection(ctx, tx, *prior)
				return e
			}
		}
		if err = fn(ctx, tx); err != nil {
			return err
		}
		var project *string
		if projectID != "" {
			project = &projectID
		}
		if err = audit.Append(ctx, tx, contracts.AuditEvent{ActorID: actor, RequestID: transport.RequestID(ctx), Action: action, RecordType: recordType, RecordID: recordID, ProjectID: project, OccurredAt: s.now()}); err != nil {
			return err
		}
		if m != nil {
			result, e := mutationResult(ctx, tx, action, recordType, recordID)
			if e != nil {
				return e
			}
			return store.Record(ctx, tx, mutationIdentity(ctx, actor, m), result)
		}
		return nil
	})
}
func (s Service) handle(capability string, fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.D.DB == nil || s.D.Clock == nil {
			transport.Unavailable(w, r)
			return
		}
		if err := (policy.Service{DB: s.D.DB, Clock: s.D.Clock}).Authorize(r.Context(), capability, ""); err != nil {
			transactions.WriteError(w, r, err)
			return
		}
		if isMutation(r) {
			var e error
			r, e = mutationRequest(w, r)
			if e != nil {
				transactions.WriteError(w, r, e)
				return
			}
		}
		fn(w, r)
	}
}
func respond(w http.ResponseWriter, r *http.Request, data any, err error) {
	if err != nil {
		transactions.WriteError(w, r, err)
		return
	}
	if replay := mutationReplay(r.Context()); replay != nil {
		data = replay
	}
	transport.JSON(w, r, 200, data)
}
func RegisterRoutes(mux *http.ServeMux, d contracts.Dependencies) {
	s := Service{D: d}
	s.registerSettings(mux)
	s.registerOverrides(mux)
	s.registerAcknowledgments(mux)
	s.registerStewardship(mux)
	s.registerOperator(mux)
}
