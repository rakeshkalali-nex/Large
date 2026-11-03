// Package contracts defines stable integration boundaries without importing any
// concrete platform, transport or domain implementation.
package contracts

import (
	"context"
	"database/sql"
	"time"
)

// Clock makes expiry, review and retention decisions deterministic in tests.
type Clock interface {
	Now() time.Time
}

// Policy uses a trusted principal in context, a closed capability and a resource
// identifier. Implementations must not derive grants from request DTO fields.
type Policy interface {
	Authorize(context.Context, string, string) error
}

// Dependencies is explicitly injected at the composition root. A nil capability
// is unavailable, never implicit permission or a successful no-op.
type Dependencies struct {
	DB           *sql.DB
	SessionKey   []byte // Private persistent CSRF/rate-limit signing key, never a bearer credential.
	CookieSecure bool
	ReadDB       *sql.DB
	Clock        Clock
	Policy       Policy
	Transactions Transactions
	Audit        AuditWriter
	Evidence     EvidenceValidator
	Costs        CostRecognition
	Booking      BookingFinance
	Projections  ReadProjections
	Operator     OperatorDiagnostics
}
