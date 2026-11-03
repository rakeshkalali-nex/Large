package contracts

import (
	"context"
	"time"
)

// MinorUnits is exact base-10 minor currency units, never binary floating point.
// Implementations validate signed int64 persistence bounds before committing.
type MinorUnits string

type SafeChange struct {
	Field  string
	Before *string
	After  *string
}

type AuditEvent struct {
	ActorID    string
	RequestID  string
	Action     string
	ProjectID  *string
	RecordType string
	RecordID   string
	OccurredAt time.Time
	Reason     *string
	Changes    []SafeChange
}

// AuditWriter validates the allowlist and rejects credentials, session/token
// secrets and internal paths. Evidence is append-only in the caller's transaction.
type AuditWriter interface {
	Append(context.Context, Transaction, AuditEvent) error
}

type EvidenceRequest struct {
	ProjectID      string
	ParentType     string
	ParentID       string
	DocumentIDs    []string
	Classification string
}

// EvidenceValidator checks active completed uploads, concrete parent links,
// matching project, MIME/classification and the most restrictive parent policy.
type EvidenceValidator interface {
	Validate(context.Context, Transaction, EvidenceRequest) error
}

type CostSource struct {
	ID               string
	ProjectID        string
	CategoryID       string
	OriginType       string
	OriginID         string
	CommitmentLineID *string
	Amount           MinorUnits
}

// CostRecognition shares an economic identity between an expense and bill;
// payments never add recognition. Source uniqueness is enforced persistently.
type CostRecognition interface {
	Recognize(context.Context, Transaction, CostSource) (string, error)
	Reverse(context.Context, Transaction, string, string) error
}

type BookingBalance struct {
	BookingID             string
	CollectedMinor        MinorUnits
	OutstandingMinor      MinorUnits
	PaidRefundsMinor       MinorUnits
	RemainingRefundable   *MinorUnits
	SourceVersions        []RecordVersion
}

type PaymentEvidence struct {
	BookingID      string
	Purpose        string
	AllocationIDs  []string
	DocumentIDs    []string
	RequiredMinor  MinorUnits
	VerifiedMinor  MinorUnits
	SourceVersions []RecordVersion
}

// BookingFinance computes from posted attributed allocations, not unapplied
// receipts. Verification is checked again inside sale/handover transactions.
type BookingFinance interface {
	Balance(context.Context, Transaction, string) (BookingBalance, error)
	Verify(context.Context, Transaction, PaymentEvidence) error
	ValidateRefund(context.Context, Transaction, string, string, MinorUnits) error
}

type ProjectionScope struct {
	ActorID       string
	AccessRevision int64
	ProjectID     *string
}

type ProjectSelector struct {
	ID             string   `json:"id"`
	Code           string   `json:"code"`
	Name           string   `json:"name"`
	Status         string   `json:"status"`
	AllowedActions []string `json:"allowedActions"`
}

type FinancialSummary struct {
	ProjectID         string     `json:"projectId"`
	ApprovedCostMinor MinorUnits `json:"approvedCostMinor"`
	PayableMinor      MinorUnits `json:"payableMinor"`
}

// ReadProjections supplies named least-data cross-area responses; it must apply
// current policy before reading, aggregating or paginating. No map[string]any
// escape hatch permits accidental delivery of confidential resource fields.
type ReadProjections interface {
	Project(context.Context, ProjectionScope, string) (ProjectSelector, error)
	FinancialSummary(context.Context, ProjectionScope, string) (FinancialSummary, error)
	BookingCollections(context.Context, ProjectionScope, string) (BookingBalance, error)
}
