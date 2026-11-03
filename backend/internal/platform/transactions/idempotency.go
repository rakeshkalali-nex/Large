package transactions

import (
	"buildestate/backend/internal/contracts"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
)

var uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Digest canonicalizes a validated typed request (not raw HTTP JSON). Exact
// numbers remain decimal strings; map key order and whitespace do not matter.
func Digest(request any) (string, error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return "", Invalid()
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if d.Decode(&v) != nil {
		return "", Invalid()
	}
	raw, err = json.Marshal(v)
	if err != nil {
		return "", Invalid()
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
func validateIdentity(i contracts.MutationIdentity) error {
	digest, err := hex.DecodeString(i.RequestDigest)
	if strings.TrimSpace(i.ActorID) == "" || strings.TrimSpace(i.RequestID) == "" || strings.TrimSpace(i.Operation) == "" || len(i.Operation) > 160 || !uuid.MatchString(i.IdempotencyKey) || err != nil || len(digest) != 32 {
		return Invalid()
	}
	return nil
}

type Store struct{ Now func() time.Time }

var _ contracts.IdempotencyStore = Store{}

// The baseline schema has no result-version column. result_id holds a versioned
// JSON reference envelope so retries retain the committed version without a
// schema change or unsafe reconstruction from a subsequently edited resource.
type reference struct {
	ID      string `json:"id"`
	Version int64  `json:"version"`
}

func (s Store) Lookup(ctx context.Context, tx contracts.Transaction, i contracts.MutationIdentity) (*contracts.CommittedResult, error) {
	if err := validateIdentity(i); err != nil {
		return nil, err
	}
	var digest, raw string
	var r contracts.CommittedResult
	err := tx.QueryRowContext(ctx, `SELECT request_digest,response_status,result_type,result_id FROM idempotency_records WHERE principal_id=? AND operation=? AND key=?`, i.ActorID, i.Operation, strings.ToLower(i.IdempotencyKey)).Scan(&digest, &r.Status, &r.ResourceType, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if digest != strings.ToLower(i.RequestDigest) {
		return nil, Conflict("IDEMPOTENCY_MISMATCH")
	}
	var ref reference
	d := json.NewDecoder(strings.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&ref) != nil || ref.ID == "" || ref.Version < 1 {
		return nil, errors.New("invalid committed reference")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return nil, errors.New("invalid committed reference")
	}
	r.ResourceID = ref.ID
	r.Version = ref.Version
	return &r, nil
}
func (s Store) Record(ctx context.Context, tx contracts.Transaction, i contracts.MutationIdentity, r contracts.CommittedResult) error {
	if err := validateIdentity(i); err != nil {
		return err
	}
	if r.Status < 200 || r.Status > 299 || r.ResourceID == "" || r.ResourceType == "" || r.Version < 1 {
		return Invalid()
	}
	raw, _ := json.Marshal(reference{r.ResourceID, r.Version})
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO idempotency_records(principal_id,operation,key,request_digest,response_status,result_type,result_id,created_at) VALUES(?,?,?,?,?,?,?,?)`, i.ActorID, i.Operation, strings.ToLower(i.IdempotencyKey), strings.ToLower(i.RequestDigest), r.Status, r.ResourceType, string(raw), Timestamp(now))
	return err
}
