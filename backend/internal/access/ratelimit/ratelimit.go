// Package ratelimit provides persistent, atomic authentication attempt budgets.
package ratelimit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"buildestate/backend/internal/contracts"
)

const (
	Login     = "login"
	Reset     = "reset"
	Issuance  = "issuance"
	timestamp = "2006-01-02T15:04:05.000Z"
)

var ErrConfiguration = errors.New("rate limiter requires a database, clock, and key of at least 32 bytes")
var ErrOperation = errors.New("unsupported rate limit operation")

type Limiter struct {
	DB    *sql.DB
	Key   []byte
	Clock contracts.Clock
}

// Attempt atomically charges BOTH budgets, including successful and throttled
// attempts. It is intentionally more conservative than counting only failures:
// callers invoke it before credential verification and never refund success.
// Login/reset allow 5 identifier and 30 origin attempts per 15 minutes; issuance
// allows 5 target and 30 issuer attempts per hour. Each subject's window begins
// on its first attempt, not at an aligned wall-clock boundary.
//
// identifier is a normalized email, reset-token digest, or issuance target ID.
// origin is a trusted peer address (without ephemeral port), or issuing user ID
// for issuance. Never pass passwords, raw tokens, or untrusted forwarded headers.
// A positive duration means deny with 429 and ceil(duration.Seconds()) Retry-After.
// Any error must fail closed. Use the platform's writable SQLite database pool.
func (l Limiter) Attempt(ctx context.Context, operation, identifier, origin string) (time.Duration, error) {
	if l.DB == nil || l.Clock == nil || len(l.Key) < 32 {
		return 0, ErrConfiguration
	}
	window := 15 * time.Minute
	switch operation {
	case Login, Reset:
	case Issuance:
		window = time.Hour
	default:
		return 0, ErrOperation
	}
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	origin = strings.TrimSpace(origin)
	subjects := []string{l.digest("identifier", identifier), l.digest("origin", origin)}
	limits := []int64{5, 30}
	tx, err := l.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	// Write first: obtains SQLite's writer lock even for a deferred transaction,
	// before reading either counter. Separate processes cannot overspend budgets.
	now := l.Clock.Now().UTC().Truncate(time.Millisecond)
	if _, err = tx.ExecContext(ctx, `DELETE FROM rate_limit_buckets WHERE operation=? AND subject_digest IN (?,?) AND expires_at<=?`, operation, subjects[0], subjects[1], now.Format(timestamp)); err != nil {
		return 0, err
	}
	var retry time.Duration
	for i, subject := range subjects {
		var start, expiry string
		var attempts int64
		err = tx.QueryRowContext(ctx, `SELECT window_start,expires_at,attempts FROM rate_limit_buckets WHERE subject_digest=? AND operation=? ORDER BY window_start DESC LIMIT 1`, subject, operation).Scan(&start, &expiry, &attempts)
		if errors.Is(err, sql.ErrNoRows) {
			start, expiry = now.Format(timestamp), now.Add(window).Format(timestamp)
			_, err = tx.ExecContext(ctx, `INSERT INTO rate_limit_buckets(subject_digest,operation,window_start,expires_at,attempts) VALUES(?,?,?,?,1)`, subject, operation, start, expiry)
			attempts = 1
		} else if err == nil {
			// Saturate after denial to avoid integer overflow during sustained abuse.
			if attempts <= limits[i] {
				attempts++
			}
			_, err = tx.ExecContext(ctx, `UPDATE rate_limit_buckets SET attempts=? WHERE subject_digest=? AND operation=? AND window_start=?`, attempts, subject, operation, start)
		}
		if err != nil {
			return 0, err
		}
		if attempts > limits[i] {
			end, parseErr := time.Parse(timestamp, expiry)
			if parseErr != nil {
				return 0, parseErr
			}
			if remaining := end.Sub(now); remaining > retry {
				retry = remaining
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return retry, nil
}

func (l Limiter) digest(kind, value string) string {
	mac := hmac.New(sha256.New, l.Key)
	mac.Write([]byte("buildestate/ratelimit/v1\x00" + kind + "\x00" + value))
	return hex.EncodeToString(mac.Sum(nil))
}
