package policy

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
)

// Principal contains only middleware-verified identifiers, never client grants.
type Principal struct {
	UserID        string
	SessionDigest string
}
type principalKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

type Service struct {
	DB    *sql.DB
	Clock contracts.Clock
}

var _ contracts.Policy = Service{}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type identity struct{ userID, role string }

func forbidden() error {
	return &transactions.Error{Status: 403, Code: "FORBIDDEN", Message: "This operation is not permitted."}
}
func expired() error {
	return &transactions.Error{Status: 401, Code: "SESSION_EXPIRED", Message: "Sign in again to continue."}
}
func unavailable() error {
	return &transactions.Error{Status: 503, Code: "SERVICE_UNAVAILABLE", Message: "The service is temporarily unavailable."}
}
func NotFound() error {
	return &transactions.Error{Status: 404, Code: "NOT_FOUND", Message: "The requested resource was not found."}
}

func (s Service) current(ctx context.Context, q queryer) (identity, error) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	if !ok || p.UserID == "" || p.SessionDigest == "" {
		return identity{}, &transactions.Error{Status: 401, Code: "AUTH_REQUIRED", Message: "Sign in to continue."}
	}
	if q == nil || s.Clock == nil {
		return identity{}, unavailable()
	}
	var role, status, idle, absolute string
	var setup int
	var access, sessionAccess, version, credential int64
	var revoked sql.NullString
	err := q.QueryRowContext(ctx, `SELECT u.role_code,u.status,u.setup_required,u.access_revision,u.version,
		s.access_revision,s.credential_revision,s.idle_expires_at,s.absolute_expires_at,s.revoked_at
		FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_digest=? AND u.id=?`, p.SessionDigest, p.UserID).
		Scan(&role, &status, &setup, &access, &version, &sessionAccess, &credential, &idle, &absolute, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return identity{}, expired()
	}
	if err != nil {
		return identity{}, unavailable()
	}
	if status != "active" || setup != 0 || revoked.Valid || access != sessionAccess || version != credential {
		return identity{}, expired()
	}
	idleAt, e1 := time.Parse(time.RFC3339Nano, idle)
	absAt, e2 := time.Parse(time.RFC3339Nano, absolute)
	now := s.Clock.Now()
	if e1 != nil || e2 != nil || !now.Before(idleAt) || !now.Before(absAt) {
		return identity{}, expired()
	}
	if len(Capabilities(role)) == 0 {
		return identity{}, forbidden()
	}
	return identity{p.UserID, role}, nil
}

// Authorize checks the route and optional project, not record ownership or
// workflow state. Use ScopeTx/RequireResourceTx for reads and the domain's
// independent-review/lifecycle checks for writes. It never refreshes idle time.
func (s Service) Authorize(ctx context.Context, capability, projectID string) error {
	if s.DB == nil {
		return unavailable()
	}
	return s.authorize(ctx, s.DB, capability, projectID)
}

// AuthorizeTx intentionally uses only the supplied transaction, never s.DB.
func (s Service) AuthorizeTx(ctx context.Context, tx contracts.Transaction, capability, projectID string) error {
	if tx == nil {
		return unavailable()
	}
	return s.authorize(ctx, tx, capability, projectID)
}

func (s Service) authorize(ctx context.Context, q queryer, capability, projectID string) error {
	i, err := s.current(ctx, q)
	if err != nil {
		return err
	}
	if !hasCapability(i.role, capability) {
		return forbidden()
	}
	// Empty scope permits a collection shell, not an unfiltered collection.
	if projectID == "" {
		return nil
	}
	var exists int
	err = q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM projects p WHERE p.id=? AND
		(?='administrator' OR EXISTS(SELECT 1 FROM user_projects up WHERE up.project_id=p.id AND up.user_id=?)))`, projectID, i.role, i.userID).Scan(&exists)
	if err != nil {
		return unavailable()
	}
	if exists != 1 {
		return forbidden()
	}
	return nil
}
