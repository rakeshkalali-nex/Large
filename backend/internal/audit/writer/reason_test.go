package writer_test

import (
	"context"
	"regexp"
	"testing"

	"buildestate/backend/internal/audit/writer"
)

func TestReasonProseAndUUID(t *testing.T) {
	db := store(t)
	tx := transaction(t, db)
	e := event()
	e.Reason = ptr("Please correct the reported quantities before resubmission.")
	if err := writer.New().Append(context.Background(), tx, e); err != nil {
		t.Fatal(err)
	}
	var id, reason string
	if err := tx.QueryRow(`SELECT id,reason FROM audit_events`).Scan(&id, &reason); err != nil {
		t.Fatal(err)
	}
	if reason != *e.Reason || !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(id) {
		t.Fatalf("invalid reason or UUID: %q %q", reason, id)
	}
	for _, unsafe := range []string{"password: hidden", "Authorization=hidden", "Bearer hidden", "panic: failed", "SQLSTATE 23000", "Traceback details", "/private/data", `C:\private\data`, "line\nsecret", `{"password":"hidden"}`, "eyJhbGciOiJIUzI1NiJ9.abc.def"} {
		e.Reason = ptr(unsafe)
		if err := writer.New().Append(context.Background(), tx, e); err != writer.ErrInvalidEvent {
			t.Fatalf("unsafe reason accepted: %v", err)
		}
	}
}
