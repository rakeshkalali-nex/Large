// Package bootstrap implements explicit, local-only initialization, never registration.
package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/audit/writer"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
)

var (
	ErrExists  = errors.New("bootstrap refused: a staff account already exists")
	ErrInvalid = errors.New("invalid bootstrap input: supply identity, company, recovery contact and instructions, and a 12 to 128 character password")
	ErrFailed  = errors.New("bootstrap failed; no initialization was committed")
)

// Input must come only from protected standard input, never arguments or logs.
type Input struct {
	DisplayName          string `json:"displayName"`
	Email                string `json:"email"`
	Password             string `json:"password"`
	CompanyName          string `json:"companyName"`
	RecoveryContactLabel string `json:"recoveryContactLabel"`
	RecoveryInstructions string `json:"recoveryInstructions"`
}

func text(value string, max int) bool {
	if !utf8.ValidString(value) || strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > max {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// Create uses the database.Open immediate writer transaction, including the
// empty-staff check, so independent operator processes cannot both initialize.
func Create(ctx context.Context, db *sql.DB, input Input) error {
	input.Email = strings.TrimSpace(input.Email)
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || !text(input.Email, 254) || !text(input.DisplayName, 200) || !text(input.CompanyName, 200) || !text(input.RecoveryContactLabel, 200) || !text(input.RecoveryInstructions, 2000) || !utf8.ValidString(input.Password) || utf8.RuneCountInString(input.Password) < 12 || utf8.RuneCountInString(input.Password) > 128 {
		return ErrInvalid
	}
	hash, err := sessions.HashPassword(input.Password)
	if err != nil {
		return ErrFailed
	}
	err = (transactions.Manager{DB: db}).WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users)`).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrExists
		}
		id, requestID := transactions.NewID(), transactions.NewID()
		if _, err := tx.ExecContext(ctx, `INSERT INTO users(id,display_name,email,normalized_email,role_code,status,password_hash,setup_required) VALUES(?,?,?,?,'administrator','active',?,0)`, id, strings.TrimSpace(input.DisplayName), input.Email, strings.ToLower(input.Email), hash); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO company_settings(id,company_name,wordmark,recovery_contact_label,recovery_instructions,created_by) VALUES('company',?,?,?,?,?)`, strings.TrimSpace(input.CompanyName), strings.TrimSpace(input.CompanyName), strings.TrimSpace(input.RecoveryContactLabel), strings.TrimSpace(input.RecoveryInstructions), id); err != nil {
			return err
		}
		audit := writer.New()
		for _, event := range []contracts.AuditEvent{
			{ActorID: id, RequestID: requestID, Action: "user.create", RecordType: "user", RecordID: id, OccurredAt: time.Now()},
			{ActorID: id, RequestID: requestID, Action: "settings.create", RecordType: "settings", RecordID: "company", OccurredAt: time.Now()},
		} {
			if err := audit.Append(ctx, tx, event); err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, ErrExists) {
		return ErrExists
	}
	if err != nil {
		return ErrFailed
	}
	return nil
}
