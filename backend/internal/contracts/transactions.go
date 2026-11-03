package contracts

import (
	"context"
	"database/sql"
)

// Transaction exposes only statements, never Commit or Rollback. Repositories
// cannot independently commit part of a business/audit/idempotency operation.
type Transaction interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Transactions runs short serialized BEGIN IMMEDIATE write transactions. The
// callback rechecks current permissions and versions and writes audit and durable
// idempotency evidence in the same transaction. An error rolls back everything.
type Transactions interface {
	WithinWrite(context.Context, func(context.Context, Transaction) error) error
}

type MutationIdentity struct {
	ActorID        string
	RequestID      string
	Operation      string
	IdempotencyKey string
	RequestDigest string
}

type RecordVersion struct {
	Resource string `json:"resource"`
	ID       string `json:"id"`
	Version  int64  `json:"version"`
}

// IdempotencyStore must run under the caller's business transaction. A replay
// still requires current authorization; conflicting digests return a conflict.
type IdempotencyStore interface {
	Lookup(context.Context, Transaction, MutationIdentity) (*CommittedResult, error)
	Record(context.Context, Transaction, MutationIdentity, CommittedResult) error
}

type CommittedResult struct {
	Status       int
	ResourceType string
	ResourceID   string
	Version      int64
}
