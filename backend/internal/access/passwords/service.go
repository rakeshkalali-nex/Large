// Package passwords owns atomic credential completion. Plaintext credentials are
// used only for verification/hashing outside the serialized writer transaction.
package passwords

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"unicode/utf8"

	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
)

type Service struct{ Sessions *sessions.Service }

func New(d contracts.Dependencies) *Service { return &Service{Sessions: sessions.New(d)} }
func InvalidToken() error {
	return &transactions.Error{Status: 422, Code: "INVALID_RESET_TOKEN", Message: "This link is invalid or expired. Contact an Administrator for identity verification and a replacement link."}
}
func invalidPassword() error {
	return &transactions.Error{Status: 422, Code: "VALIDATION_FAILED", Message: "Use a password containing 12 to 128 characters."}
}

var keyPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func ValidKey(key string) bool { return keyPattern.MatchString(key) }

// Reserve rejects replays rather than persisting/reconstructing secret responses.
// Its digest describes only a public operation and record, never a password or link.
func Reserve(ctx context.Context, tx contracts.Transaction, actor, operation, record, key string) error {
	digest, _ := transactions.Digest(struct{ Operation, Record string }{operation, record})
	requestID := transport.RequestID(ctx)
	if requestID == "" {
		requestID = transactions.NewID()
	}
	id := contracts.MutationIdentity{ActorID: actor, Operation: operation, RequestID: requestID, IdempotencyKey: key, RequestDigest: digest}
	store := transactions.Store{}
	old, err := store.Lookup(ctx, tx, id)
	if err != nil {
		return err
	}
	if old != nil {
		return transactions.Conflict("IDEMPOTENCY_MISMATCH")
	}
	return store.Record(ctx, tx, id, contracts.CommittedResult{Status: 200, ResourceType: "user", ResourceID: record, Version: 1})
}

func (s *Service) Reset(ctx context.Context, token, password, key string) error {
	if !ValidKey(key) {
		return transactions.Invalid()
	}
	if len(token) != 43 {
		return InvalidToken()
	}
	// Reject invalid tokens before expensive hashing, then recheck under the writer lock.
	var exists int
	err := s.Sessions.DB.QueryRowContext(ctx, `SELECT count(*) FROM password_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_digest=? AND t.consumed_at IS NULL AND t.revoked_at IS NULL AND t.expires_at>? AND u.status IN ('active','setup-required')`, sessions.Digest(token), transactions.Timestamp(s.Sessions.Clock.Now())).Scan(&exists)
	if err != nil {
		return err
	}
	if exists != 1 {
		return InvalidToken()
	}
	hash, err := sessions.HashPassword(password)
	if err != nil {
		return invalidPassword()
	}
	return (transactions.Manager{DB: s.Sessions.DB}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		var user, kind, status string
		var setup int
		now := transactions.Timestamp(s.Sessions.Clock.Now())
		err := tx.QueryRowContext(ctx, `SELECT t.user_id,t.kind,u.status,u.setup_required FROM password_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_digest=? AND t.consumed_at IS NULL AND t.revoked_at IS NULL AND t.expires_at>? AND u.status IN ('active','setup-required')`, sessions.Digest(token), now).Scan(&user, &kind, &status, &setup)
		if errors.Is(err, sql.ErrNoRows) {
			return InvalidToken()
		}
		if err != nil {
			return err
		}
		if kind == "setup" && status != "setup-required" && setup == 0 {
			return InvalidToken()
		}
		if err = Reserve(ctx, tx, user, "auth.reset-password", user, key); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE password_tokens SET consumed_at=? WHERE token_digest=?`, now, sessions.Digest(token)); err != nil {
			return err
		}
		action := "auth.password.reset.completed"
		if kind == "setup" {
			action = "auth.setup.completed"
		}
		return s.complete(ctx, tx, user, hash, action)
	})
}

func (s *Service) Change(ctx context.Context, session, current, password, key string) error {
	if !ValidKey(key) {
		return transactions.Invalid()
	}
	identity, err := s.Sessions.Restore(ctx, session)
	if err != nil {
		return err
	}
	if identity.SetupRequired {
		return &transactions.Error{Status: 403, Code: "SETUP_REQUIRED", Message: "Complete password setup using your setup link."}
	}
	if !utf8.ValidString(current) || utf8.RuneCountInString(current) > 128 {
		return invalidPassword()
	}
	var old string
	if err = s.Sessions.DB.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id=?`, identity.User.ID).Scan(&old); err != nil {
		return err
	}
	if !sessions.VerifyPassword(old, current) {
		return &transactions.Error{Status: 401, Code: "INVALID_CREDENTIALS", Message: "Current password is incorrect."}
	}
	hash, err := sessions.HashPassword(password)
	if err != nil {
		return invalidPassword()
	}
	return s.Sessions.WithinAuthorizedWrite(ctx, session, "auth.password.change", "", func(ctx context.Context, tx contracts.Transaction) error {
		var latest string
		if err := tx.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id=?`, identity.User.ID).Scan(&latest); err != nil {
			return err
		}
		if latest != old {
			return &transactions.Error{Status: 401, Code: "SESSION_EXPIRED", Message: "Sign in again."}
		}
		if err := Reserve(ctx, tx, identity.User.ID, "auth.change-password", identity.User.ID, key); err != nil {
			return err
		}
		return s.complete(ctx, tx, identity.User.ID, hash, "auth.password.changed")
	})
}

func (s *Service) complete(ctx context.Context, tx contracts.Transaction, user, hash, action string) error {
	now := transactions.Timestamp(s.Sessions.Clock.Now())
	if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash=?,setup_required=0,status='active',access_revision=access_revision+1,version=version+1,updated_at=? WHERE id=?`, hash, now, user); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL`, now, user); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE password_tokens SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL AND consumed_at IS NULL`, now, user); err != nil {
		return err
	}
	request := transport.RequestID(ctx)
	if request == "" {
		request = transactions.NewID()
	}
	return s.Sessions.Audit.Append(ctx, tx, contracts.AuditEvent{ActorID: user, RequestID: request, RecordID: user, RecordType: "user", Action: action, OccurredAt: s.Sessions.Clock.Now()})
}
