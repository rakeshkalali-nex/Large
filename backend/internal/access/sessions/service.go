// Package sessions owns opaque, persisted authentication and its minimal client contract.
package sessions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/audit/writer"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/clock"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
)

const CookieName = "buildestate_session"
const IdleLifetime = 30 * time.Minute
const AbsoluteLifetime = 12 * time.Hour
const timestamp = "2006-01-02T15:04:05.000Z"

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}
type Identity struct {
	SetupRequired     bool     `json:"setupRequired"`
	User              User     `json:"user"`
	ProjectIDs        []string `json:"projectIds"`
	Capabilities      []string `json:"capabilities"`
	LandingPath       string   `json:"landingPath"`
	AccessRevision    int64    `json:"accessRevision"`
	IdleExpiresAt     string   `json:"idleExpiresAt"`
	AbsoluteExpiresAt string   `json:"absoluteExpiresAt"`
	CSRFToken         string   `json:"csrfToken,omitempty"`
}
type Service struct {
	DB    *sql.DB
	Clock contracts.Clock
	Audit contracts.AuditWriter
}

func New(d contracts.Dependencies) *Service {
	c := d.Clock
	if c == nil {
		c = clock.UTC{}
	}
	a := d.Audit
	if a == nil {
		a = writer.New()
	}
	return &Service{DB: d.DB, Clock: c, Audit: a}
}
func Digest(token string) string { h := sha256.Sum256([]byte(token)); return hex.EncodeToString(h[:]) }
func randomToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b), err
}
func denied() error {
	return &transactions.Error{Status: 401, Code: "INVALID_CREDENTIALS", Message: "Email or password is incorrect."}
}
func expired() error {
	return &transactions.Error{Status: 401, Code: "SESSION_EXPIRED", Message: "Your session has expired. Sign in again."}
}
func (s *Service) event(ctx context.Context, tx contracts.Transaction, actor, action string) error {
	id := transport.RequestID(ctx)
	if id == "" {
		id = transactions.NewID()
	}
	return s.Audit.Append(ctx, tx, contracts.AuditEvent{ActorID: actor, RequestID: id, Action: action, RecordType: "authentication", RecordID: transactions.NewID(), OccurredAt: s.Clock.Now()})
}

func (s *Service) Login(ctx context.Context, email, password, priorToken string) (string, Identity, error) {
	var id, status string
	var hash sql.NullString
	var revision, version int64
	err := s.DB.QueryRowContext(ctx, `SELECT id,password_hash,status,access_revision,version FROM users WHERE normalized_email=?`, strings.ToLower(strings.TrimSpace(email))).Scan(&id, &hash, &status, &revision, &version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", Identity{}, err
	}
	verified := VerifyPassword(hash.String, password)
	if !verified || (status != "active" && status != "setup-required") {
		err = (transactions.Manager{DB: s.DB}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
			return s.event(ctx, tx, "", "auth.login.failed")
		})
		if err != nil {
			return "", Identity{}, err
		}
		return "", Identity{}, denied()
	}
	token, err := randomToken()
	if err != nil {
		return "", Identity{}, err
	}
	now := s.Clock.Now().UTC()
	var identity Identity
	err = (transactions.Manager{DB: s.DB}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		// Hashing is outside the writer; recheck exactly what was verified inside it.
		var current int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE id=? AND password_hash=? AND access_revision=? AND version=? AND status IN ('active','setup-required')`, id, hash.String, revision, version).Scan(&current); err != nil {
			return err
		}
		if current != 1 {
			return denied()
		}
		if priorToken != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE sessions SET revoked_at=? WHERE token_digest=? AND revoked_at IS NULL`, now.Format(timestamp), Digest(priorToken)); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO sessions(id,token_digest,user_id,access_revision,credential_revision,created_at,last_interaction_at,idle_expires_at,absolute_expires_at) VALUES(?,?,?,?,?,?,?,?,?)`, transactions.NewID(), Digest(token), id, revision, version, now.Format(timestamp), now.Format(timestamp), now.Add(IdleLifetime).Format(timestamp), now.Add(AbsoluteLifetime).Format(timestamp))
		if err != nil {
			return err
		}
		identity, err = s.restore(ctx, tx, token, false)
		if err != nil {
			return err
		}
		return s.event(ctx, tx, id, "auth.login.succeeded")
	})
	return token, identity, err
}

// Restore never extends idle expiry. Only explicit authorized interactions do.
func (s *Service) Restore(ctx context.Context, token string) (Identity, error) {
	identity, err := s.restore(ctx, s.DB, token, false)
	var safe *transactions.Error
	if errors.As(err, &safe) && safe.Code == "SESSION_EXPIRED" {
		if auditErr := s.expire(ctx, token); auditErr != nil {
			return Identity{}, auditErr
		}
	}
	return identity, err
}

func (s *Service) expire(ctx context.Context, token string) error {
	return (transactions.Manager{DB: s.DB}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		var actor string
		now := s.Clock.Now().UTC().Format(timestamp)
		err := tx.QueryRowContext(ctx, `SELECT user_id FROM sessions WHERE token_digest=? AND revoked_at IS NULL AND (idle_expires_at<=? OR absolute_expires_at<=?)`, Digest(token), now, now).Scan(&actor)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE sessions SET revoked_at=? WHERE token_digest=?`, now, Digest(token)); err != nil {
			return err
		}
		return s.event(ctx, tx, actor, "auth.session.expired")
	})
}

// Touch is called only after a successful explicitly interactive response, never
// for restoration or background polling. It cannot resurrect an expired session.
func (s *Service) Touch(ctx context.Context, token string) error {
	return (transactions.Manager{DB: s.DB}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		_, err := s.restore(ctx, tx, token, true)
		return err
	})
}
func (s *Service) restore(ctx context.Context, q contracts.Transaction, token string, touch bool) (Identity, error) {
	var out Identity
	out.ProjectIDs = []string{}
	out.Capabilities = []string{}
	if token == "" {
		return out, &transactions.Error{Status: 401, Code: "AUTH_REQUIRED", Message: "Sign in to continue."}
	}
	if len(token) != 43 {
		return out, expired()
	}
	var status string
	var setup int
	err := q.QueryRowContext(ctx, `SELECT u.id,u.display_name,u.email,u.role_code,u.status,u.setup_required,u.access_revision,s.idle_expires_at,s.absolute_expires_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_digest=? AND s.revoked_at IS NULL AND s.access_revision=u.access_revision AND s.credential_revision=u.version AND u.status IN ('active','setup-required')`, Digest(token)).Scan(&out.User.ID, &out.User.DisplayName, &out.User.Email, &out.User.Role, &status, &setup, &out.AccessRevision, &out.IdleExpiresAt, &out.AbsoluteExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return out, expired()
	}
	if err != nil {
		return out, err
	}
	now := s.Clock.Now().UTC()
	idle, e1 := time.Parse(timestamp, out.IdleExpiresAt)
	absolute, e2 := time.Parse(timestamp, out.AbsoluteExpiresAt)
	if e1 != nil || e2 != nil || !now.Before(idle) || !now.Before(absolute) {
		return Identity{}, expired()
	}
	out.LandingPath = policy.LandingPath(out.User.Role)
	if status == "setup-required" || setup != 0 {
		out.SetupRequired = true
		out.LandingPath = "/login"
		return out, nil
	}
	out.Capabilities = policy.Capabilities(out.User.Role)
	query := `SELECT project_id FROM user_projects WHERE user_id=? ORDER BY project_id`
	args := []any{out.User.ID}
	if out.User.Role == "administrator" {
		query = `SELECT id FROM projects ORDER BY id`
		args = nil
	}
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return out, err
		}
		out.ProjectIDs = append(out.ProjectIDs, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	if touch {
		next := now.Add(IdleLifetime)
		if next.After(absolute) {
			next = absolute
		}
		out.IdleExpiresAt = next.Format(timestamp)
		_, err = q.ExecContext(ctx, `UPDATE sessions SET last_interaction_at=?,idle_expires_at=? WHERE token_digest=? AND revoked_at IS NULL`, now.Format(timestamp), out.IdleExpiresAt, Digest(token))
	}
	return out, err
}

// WithinAuthorizedWrite is the domain mutation boundary: authorization and session
// liveness are checked again in the same serialized transaction as the writes.
func (s *Service) WithinAuthorizedWrite(ctx context.Context, token, capability, projectID string, fn func(context.Context, contracts.Transaction) error) error {
	return (transactions.Manager{DB: s.DB}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		identity, err := s.restore(ctx, tx, token, true)
		if err != nil {
			return err
		}
		ctx = policy.WithPrincipal(ctx, policy.Principal{UserID: identity.User.ID, SessionDigest: Digest(token)})
		if err := (policy.Service{DB: s.DB, Clock: s.Clock}).AuthorizeTx(ctx, tx, capability, projectID); err != nil {
			return err
		}
		return fn(ctx, tx)
	})
}
func (s *Service) Logout(ctx context.Context, token string) error {
	return (transactions.Manager{DB: s.DB}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		var actor string
		err := tx.QueryRowContext(ctx, `SELECT user_id FROM sessions WHERE token_digest=? AND revoked_at IS NULL`, Digest(token)).Scan(&actor)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE sessions SET revoked_at=? WHERE token_digest=?`, s.Clock.Now().UTC().Format(timestamp), Digest(token)); err != nil {
			return err
		}
		return s.event(ctx, tx, actor, "auth.logout")
	})
}
