// Package writer appends deliberately limited audit evidence in the caller's transaction.
package writer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"buildestate/backend/internal/contracts"
)

// ErrInvalidEvent never includes rejected input (which may contain credentials).
var ErrInvalidEvent = errors.New("invalid audit event")

// ErrAppend never exposes database diagnostics or internal paths.
var ErrAppend = errors.New("audit append failed")

// Writer is stateless. Its zero value is ready for use and never owns a database.
type Writer struct{}

var _ contracts.AuditWriter = (*Writer)(nil)

// New returns the default fail-closed writer. No configuration can bypass its allowlists.
func New() *Writer { return &Writer{} }

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

func member(value, choices string) bool {
	for _, choice := range strings.Fields(choices) {
		if value == choice {
			return true
		}
	}
	return false
}

func recordType(value string) bool {
	if value == "site-log" {
		return true
	}
	return member(value, "user project site milestone task daily-log inspection inspection-item defect material supplier cost-category purchase-order purchase-order-amendment goods-receipt stock-transaction stock-adjustment contractor work-order work-order-amendment budget budget-revision expense property lead customer site-visit reservation booking booking-amendment cancellation invoice payment refund handover document upload-batch settings reference-value property-type privacy-request legal-hold retention-review reconciliation authentication")
}

func authentication(action string) bool {
	return member(action, "auth.login.succeeded auth.login.failed auth.logout auth.session.expired auth.password.changed auth.password.reset.requested auth.password.reset.completed auth.setup.completed auth.access.denied")
}

func validAction(action, resource string) bool {
	if authentication(action) {
		return resource == "authentication" || resource == "user"
	}
	if resource == "authentication" {
		return false
	}
	verb := strings.TrimPrefix(action, resource+".")
	return member(verb, "create update archive restore submit return approve reject post reverse cancel complete activate deactivate assign unassign confirm release execute") && (action == verb || action == resource+"."+verb)
}

var unsafeReason = regexp.MustCompile(`(?i)(password|passwd|secret|token|cookie|authorization|api[_ -]?key|session[_ -]?id)\s*[:=]|\bbearer\s+|\beyJ[A-Za-z0-9_-]*\.|-----BEGIN|\bpanic:|\b(sqlite|sqlstate|stack trace|traceback|goroutine\s+[0-9]|sql error)\b|[A-Za-z0-9_-]{40,}`)

// Reasons are bounded human prose, never request dumps or internal diagnostics.
// Callers remain responsible for not supplying secrets: no generic prose filter
// can prove absence of an arbitrary credential. Structured changes stay allowlisted.
func validReason(reason string) bool {
	if !utf8.ValidString(reason) || len(reason) > 1000 || strings.TrimSpace(reason) == "" || unsafeReason.MatchString(reason) {
		return false
	}
	for _, r := range reason {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || strings.ContainsRune(`/\{}[]`, r) {
			return false
		}
	}
	return true
}

type change struct {
	Field  string  `json:"field"`
	Before *string `json:"before"`
	After  *string `json:"after"`
}

// Changes accept only named enums, canonical integers, booleans and dates.
// In particular no free-text/name/notes/error/path/token field is serializable.
func validValue(field string, value *string) bool {
	if value == nil {
		return knownField(field)
	}
	v := *value
	switch field {
	case "status":
		return member(v, "draft active archived setup-required deactivated on-hold completed planned in-progress cancelled submitted returned approved rejected posted reversed part-received received superseded revised scheduled pending-review converted expired cancellation-requested unconfirmed confirmed open resolved reopened quarantine validated committed failed in-review reviewed executed released")
	case "role_code":
		return member(v, "administrator project-manager site-engineer procurement-officer finance-officer sales-executive management-viewer")
	case "classification":
		return member(v, "operational financial sales restricted")
	case "method":
		return member(v, "cash cheque bank-transfer other")
	case "priority":
		return member(v, "low normal high critical")
	case "archived", "enabled":
		return v == "true" || v == "false"
	case "start_date", "end_date", "date", "due_date":
		d, err := time.Parse("2006-01-02", v)
		return err == nil && d.Format("2006-01-02") == v
	case "version", "revision", "access_revision", "amount_minor", "total_minor", "quantity", "quantity_delta", "completion_bps", "weight_bps":
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || strconv.FormatInt(n, 10) != v {
			return false
		}
		switch field {
		case "version", "revision", "access_revision":
			return n > 0
		case "completion_bps":
			return n >= 0 && n <= 10000
		case "weight_bps":
			return n > 0 && n <= 10000
		case "quantity_delta":
			return n != 0
		default:
			return n >= 0
		}
	}
	return false
}

func knownField(field string) bool {
	return member(field, "status role_code classification method priority archived enabled start_date end_date date due_date version revision access_revision amount_minor total_minor quantity quantity_delta completion_bps weight_bps")
}

// Append executes one INSERT on tx; it never begins, commits or rolls back a transaction.
// An empty ActorID is SQL NULL only for an explicitly recognized authentication action.
// Callers must supply a transaction, not a connection pool, as required by contracts.
func (Writer) Append(ctx context.Context, tx contracts.Transaction, event contracts.AuditEvent) error {
	if tx == nil || (reflect.ValueOf(tx).Kind() == reflect.Ptr && reflect.ValueOf(tx).IsNil()) {
		return ErrInvalidEvent
	}
	if !identifier.MatchString(event.RequestID) || !identifier.MatchString(event.RecordID) || !recordType(event.RecordType) || !validAction(event.Action, event.RecordType) || event.OccurredAt.IsZero() {
		return ErrInvalidEvent
	}
	stamp := event.OccurredAt.UTC()
	if stamp.Year() < 1 || stamp.Year() > 9999 || (event.ActorID == "" && !authentication(event.Action)) || (event.ActorID != "" && !identifier.MatchString(event.ActorID)) {
		return ErrInvalidEvent
	}
	if event.ProjectID != nil && !identifier.MatchString(*event.ProjectID) || event.Reason != nil && !validReason(*event.Reason) || len(event.Changes) > 32 {
		return ErrInvalidEvent
	}
	changes := make([]change, 0, len(event.Changes))
	seen := make(map[string]bool, len(event.Changes))
	for _, c := range event.Changes {
		if seen[c.Field] || c.Before == nil && c.After == nil || !validValue(c.Field, c.Before) || !validValue(c.Field, c.After) {
			return ErrInvalidEvent
		}
		seen[c.Field] = true
		changes = append(changes, change{Field: c.Field, Before: c.Before, After: c.After})
	}
	payload, err := json.Marshal(changes)
	if err != nil {
		return ErrInvalidEvent
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return ErrAppend
	}
	random[6] = (random[6] & 0x0f) | 0x40
	random[8] = (random[8] & 0x3f) | 0x80
	h := hex.EncodeToString(random[:])
	id := h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
	var actor any
	if event.ActorID != "" {
		actor = event.ActorID
	}
	retention := "business"
	if authentication(event.Action) {
		retention = "authentication"
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_events
		(id,actor_id,occurred_at,action,project_id,record_type,record_id,request_id,reason,changes_json,retention_class)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`, id, actor, stamp.Format("2006-01-02T15:04:05.000Z"), event.Action, event.ProjectID, event.RecordType, event.RecordID, event.RequestID, event.Reason, string(payload), retention)
	if err != nil {
		return ErrAppend
	}
	return nil
}
