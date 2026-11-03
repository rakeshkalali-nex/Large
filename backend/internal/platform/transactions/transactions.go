// Package transactions owns atomic service boundaries, not domain decisions.
package transactions

import (
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transport"
	"context"
	"database/sql"
	"errors"
	"net/http"
)

// Error contains only stable public information, never driver messages.
type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string { return e.Code }
func Conflict(code string) error {
	return &Error{409, code, "The record changed or this action is not permitted in its current state."}
}
func Invalid() error { return &Error{422, "VALIDATION_FAILED", "The supplied mutation is invalid."} }
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var safe *Error
	if errors.As(err, &safe) {
		transport.Error(w, r, safe.Status, safe.Code, safe.Message)
		return
	}
	transport.Error(w, r, 500, "INTERNAL_ERROR", "The operation could not be completed.")
}
func storageError(err error) error {
	if err == nil {
		return nil
	}
	var coded interface{ Code() int }
	if errors.As(err, &coded) && (coded.Code()&255 == 5 || coded.Code()&255 == 6) {
		return &Error{503, "STORAGE_BUSY", "Storage is busy. Retry deliberately with the same request key."}
	}
	return err
}

type Manager struct{ DB *sql.DB }

var _ contracts.Transactions = Manager{}

// database.Open configures the driver's BEGIN IMMEDIATE and serialized writer.
// Only the restricted statement wrapper is exposed, including under type assertion.
func (m Manager) WithinWrite(ctx context.Context, fn func(context.Context, contracts.Transaction) error) error {
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return storageError(err)
	}
	defer tx.Rollback() // also executes on panic and cancelled context
	if err = fn(ctx, statements{tx}); err != nil {
		return storageError(err)
	}
	return storageError(tx.Commit())
}

type statements struct{ tx *sql.Tx }

func (s statements) ExecContext(c context.Context, q string, a ...any) (sql.Result, error) {
	return s.tx.ExecContext(c, q, a...)
}
func (s statements) QueryContext(c context.Context, q string, a ...any) (*sql.Rows, error) {
	return s.tx.QueryContext(c, q, a...)
}
func (s statements) QueryRowContext(c context.Context, q string, a ...any) *sql.Row {
	return s.tx.QueryRowContext(c, q, a...)
}

// Service makes current authorization mandatory even on replay. Work must write
// business effects and return safe audit evidence; no result escapes before commit.
type Service struct {
	Transactions contracts.Transactions
	Idempotency  contracts.IdempotencyStore
	Audit        contracts.AuditWriter
}

func (s Service) Mutate(ctx context.Context, id contracts.MutationIdentity, authorize func(context.Context, contracts.Transaction) error, work func(context.Context, contracts.Transaction) (contracts.CommittedResult, contracts.AuditEvent, error)) (contracts.CommittedResult, error) {
	var result contracts.CommittedResult
	if authorize == nil || work == nil || s.Transactions == nil || s.Idempotency == nil || s.Audit == nil {
		return result, Invalid()
	}
	err := s.Transactions.WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		if err := authorize(ctx, tx); err != nil {
			return err
		}
		old, err := s.Idempotency.Lookup(ctx, tx, id)
		if err != nil {
			return err
		}
		if old != nil {
			result = *old
			return nil
		}
		next, event, err := work(ctx, tx)
		if err != nil {
			return err
		}
		if event.ActorID != id.ActorID || event.RequestID != id.RequestID || event.Action != id.Operation || event.RecordID != next.ResourceID || event.RecordType != next.ResourceType {
			return Invalid()
		}
		if err = s.Audit.Append(ctx, tx, event); err != nil {
			return err
		}
		if err = s.Idempotency.Record(ctx, tx, id, next); err != nil {
			return err
		}
		result = next
		return nil
	})
	if err != nil {
		return contracts.CommittedResult{}, err
	}
	return result, nil
}
