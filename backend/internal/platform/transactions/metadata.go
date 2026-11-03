package transactions

import (
	"crypto/rand"
	"fmt"
	"math"
	"strings"
	"time"
)

const TimestampLayout = "2006-01-02T15:04:05.000Z"

func Timestamp(t time.Time) string { return t.UTC().Format(TimestampLayout) }
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = b[6]&15 | 64
	b[8] = b[8]&63 | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

type Metadata struct {
	ID        string  `json:"id"`
	CreatedBy string  `json:"createdBy"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
	Version   int64   `json:"version"`
	ProjectID *string `json:"projectId"`
}

func NewMetadata(actor string, project *string, now time.Time) (Metadata, error) {
	if strings.TrimSpace(actor) == "" || now.IsZero() || project != nil && strings.TrimSpace(*project) == "" {
		return Metadata{}, Invalid()
	}
	var p *string
	if project != nil {
		v := *project
		p = &v
	}
	return Metadata{NewID(), actor, Timestamp(now), Timestamp(now), 1, p}, nil
}
func CheckVersion(current, expected int64) error {
	if expected < 1 || current != expected || current == math.MaxInt64 {
		return Conflict("STALE_VERSION")
	}
	return nil
}
func (m Metadata) Advance(expected int64, now time.Time) (Metadata, error) {
	if err := CheckVersion(m.Version, expected); err != nil {
		return m, err
	}
	if now.IsZero() || Timestamp(now) < m.UpdatedAt {
		return m, Invalid()
	}
	m.Version++
	m.UpdatedAt = Timestamp(now)
	return m, nil
}

// Lifecycle edges are explicitly supplied by the owning workflow, never from HTTP input.
type Edge struct {
	From, To       string
	ReasonRequired bool
}

func CheckTransition(from, to, reason string, edges []Edge) error {
	for _, e := range edges {
		if e.From == from && e.To == to {
			if e.ReasonRequired && strings.TrimSpace(reason) == "" {
				return Invalid()
			}
			return nil
		}
	}
	return Conflict("INVALID_TRANSITION")
}

// Retirement validates a retaining lifecycle action. It never issues DELETE.
func CheckRetirement(from, to, reason string, edges []Edge) error {
	if to != "archived" && to != "cancelled" && to != "reversed" {
		return Conflict("INVALID_TRANSITION")
	}
	if strings.TrimSpace(reason) == "" {
		return Invalid()
	}
	return CheckTransition(from, to, reason, edges)
}
