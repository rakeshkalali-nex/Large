// Package recovery issues private, single-display setup and recovery links.
package recovery

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"buildestate/backend/internal/access/passwords"
	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/access/ratelimit"
	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
)

const SetupLifetime = 24 * time.Hour
const RecoveryLifetime = 30 * time.Minute

type Input struct {
	Version          int64  `json:"version"`
	Purpose          string `json:"purpose"`
	IdentityVerified bool   `json:"identityVerified"`
	Reason           string `json:"reason"`
}
type Result struct {
	UserID    string `json:"userId"`
	ResetLink string `json:"resetLink"`
	ExpiresAt string `json:"expiresAt"`
}
type Guidance struct {
	Instructions string `json:"instructions"`
	ContactLabel string `json:"contactLabel"`
}
type Service struct {
	Sessions *sessions.Service
	Limits   ratelimit.Limiter
}

func New(d contracts.Dependencies) *Service {
	s := sessions.New(d)
	return &Service{s, ratelimit.Limiter{DB: d.DB, Clock: s.Clock, Key: d.SessionKey}}
}
func (s *Service) Guidance(ctx context.Context) (Guidance, error) {
	g := Guidance{Instructions: "Contact an Administrator for identity verification. They will deliver a single-use reset link through your company's trusted private channel. If a link has expired or was used, request a replacement.", ContactLabel: "Administrator"}
	var instructions, label string
	err := s.Sessions.DB.QueryRowContext(ctx, `SELECT recovery_instructions,recovery_contact_label FROM company_settings WHERE id='company'`).Scan(&instructions, &label)
	if errors.Is(err, sql.ErrNoRows) {
		return g, nil
	}
	if err != nil {
		return Guidance{}, err
	}
	if strings.TrimSpace(instructions) != "" {
		g.Instructions = instructions + " " + g.Instructions
	}
	if strings.TrimSpace(label) != "" {
		g.ContactLabel = label
	}
	return g, nil
}
func (s *Service) Issue(ctx context.Context, session, target, key string, in Input) (Result, time.Duration, error) {
	if !passwords.ValidKey(key) || in.Version < 1 || (in.Purpose != "setup" && in.Purpose != "recovery") || strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 1000 || !in.IdentityVerified {
		return Result{}, 0, transactions.Invalid()
	}
	actor, err := s.Sessions.Restore(ctx, session)
	if err != nil {
		return Result{}, 0, err
	}
	if actor.SetupRequired || actor.User.Role != "administrator" {
		return Result{}, 0, &transactions.Error{Status: 403, Code: "FORBIDDEN", Message: "Only an Administrator may issue a setup or recovery link."}
	}
	wait, err := s.Limits.Attempt(ctx, ratelimit.Issuance, target, actor.User.ID)
	if err != nil {
		return Result{}, 0, err
	}
	if wait > 0 {
		return Result{}, wait, &transactions.Error{Status: 429, Code: "RATE_LIMITED", Message: "Too many attempts. Try again later."}
	}
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return Result{}, 0, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	var out Result
	err = s.Sessions.WithinAuthorizedWrite(ctx, session, "users.manage", "", func(ctx context.Context, tx contracts.Transaction) error {
		var status string
		var setup int
		var version int64
		err := tx.QueryRowContext(ctx, `SELECT status,setup_required,version FROM users WHERE id=?`, target).Scan(&status, &setup, &version)
		if errors.Is(err, sql.ErrNoRows) {
			return policy.NotFound()
		}
		if err != nil {
			return err
		}
		if err = transactions.CheckVersion(version, in.Version); err != nil {
			return err
		}
		if status == "deactivated" || in.Purpose == "setup" && (setup == 0 && status != "setup-required") {
			return transactions.Conflict("INVALID_TRANSITION")
		}
		if err = passwords.Reserve(ctx, tx, actor.User.ID, "users.password-reset", target, key); err != nil {
			return err
		}
		now := s.Sessions.Clock.Now()
		life := RecoveryLifetime
		if in.Purpose == "setup" {
			life = SetupLifetime
		}
		expiry := transactions.Timestamp(now.Add(life))
		stamp := transactions.Timestamp(now)
		if _, err = tx.ExecContext(ctx, `UPDATE password_tokens SET revoked_at=? WHERE user_id=? AND kind=? AND consumed_at IS NULL AND revoked_at IS NULL`, stamp, target, in.Purpose); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO password_tokens(id,token_digest,user_id,kind,issuer_id,created_at,expires_at,identity_verified,reason) VALUES(?,?,?,?,?,?,?,?,?)`, transactions.NewID(), sessions.Digest(token), target, in.Purpose, actor.User.ID, stamp, expiry, 1, in.Reason); err != nil {
			return err
		}
		request := transport.RequestID(ctx)
		if request == "" {
			request = transactions.NewID()
		}
		// The audit writer's reason allowlist rejects credential-shaped free text.
		if err = s.Sessions.Audit.Append(ctx, tx, contracts.AuditEvent{ActorID: actor.User.ID, RequestID: request, RecordID: target, RecordType: "user", Action: "auth.password.reset.requested", Reason: &in.Reason, OccurredAt: now}); err != nil {
			return err
		}
		out = Result{UserID: target, ResetLink: "/login#token=" + token, ExpiresAt: expiry}
		return nil
	})
	if err != nil {
		return Result{}, 0, err
	}
	return out, 0, nil
}
