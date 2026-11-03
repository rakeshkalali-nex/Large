package writer_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"buildestate/backend/internal/audit/writer"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/testsupport"
)

func ptr(s string) *string { return &s }

func event() contracts.AuditEvent {
	return contracts.AuditEvent{
		ActorID: "actor-1", RequestID: "request-1", Action: "user.update",
		RecordType: "user", RecordID: "actor-1",
		OccurredAt: time.Date(2026, 11, 3, 15, 0, 1, 123456789, time.FixedZone("local", 19800)),
		Reason:     ptr("correction"),
		Changes:    []contracts.SafeChange{{Field: "status", Before: ptr("setup-required"), After: ptr("active")}},
	}
}

func store(t *testing.T) *sql.DB {
	t.Helper()
	db := testsupport.Database(t)
	if _, err := db.Exec(`INSERT INTO users(id,display_name,email,normalized_email,role_code) VALUES ('actor-1','Actor','actor@example.invalid','actor@example.invalid','administrator')`); err != nil {
		t.Fatal(err)
	}
	return db
}

func transaction(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	return tx
}

func count(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM audit_events`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestAppendMetadataUTCAndImmutable(t *testing.T) {
	db := store(t)
	tx := transaction(t, db)
	w := writer.New()
	for range 2 {
		if err := w.Append(context.Background(), tx, event()); err != nil {
			t.Fatal(err)
		}
	}
	var actor, stamp, action, record, recordID, request, reason, changes, retention string
	var project, digest sql.NullString
	if err := tx.QueryRow(`SELECT actor_id,occurred_at,action,record_type,record_id,request_id,reason,changes_json,retention_class,project_id,origin_digest FROM audit_events LIMIT 1`).Scan(&actor, &stamp, &action, &record, &recordID, &request, &reason, &changes, &retention, &project, &digest); err != nil {
		t.Fatal(err)
	}
	if actor != "actor-1" || stamp != "2026-11-03T09:30:01.123Z" || action != "user.update" || record != "user" || recordID != "actor-1" || request != "request-1" || reason != "correction" || retention != "business" || project.Valid || digest.Valid {
		t.Fatalf("unexpected metadata: %q %q %q %q %q %q %q %q", actor, stamp, action, record, recordID, request, reason, retention)
	}
	if changes != `[{"field":"status","before":"setup-required","after":"active"}]` {
		t.Fatalf("unexpected changes: %s", changes)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if count(t, db) != 2 {
		t.Fatal("append must create independent rows")
	}
	for _, statement := range []string{`UPDATE audit_events SET action='delete'`, `DELETE FROM audit_events`} {
		if _, err := db.Exec(statement); err == nil {
			t.Fatal("audit history was mutable")
		}
	}
	if count(t, db) != 2 {
		t.Fatal("immutable history changed")
	}
}

func TestRollbackIncludesBusinessAndAudit(t *testing.T) {
	db := store(t)
	tx := transaction(t, db)
	if _, err := tx.Exec(`UPDATE users SET display_name='Changed' WHERE id='actor-1'`); err != nil {
		t.Fatal(err)
	}
	if err := (writer.Writer{}).Append(context.Background(), tx, event()); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := db.QueryRow(`SELECT display_name FROM users WHERE id='actor-1'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Actor" || count(t, db) != 0 {
		t.Fatal("business and audit must roll back together")
	}
}

func TestAnonymousAuthentication(t *testing.T) {
	db := store(t)
	tx := transaction(t, db)
	e := event()
	e.ActorID, e.Action, e.RecordType = "", "auth.login.failed", "authentication"
	e.Reason, e.Changes = ptr("invalid-credentials"), nil
	if err := writer.New().Append(context.Background(), tx, e); err != nil {
		t.Fatal(err)
	}
	var actor sql.NullString
	var retention, changes string
	if err := tx.QueryRow(`SELECT actor_id,retention_class,changes_json FROM audit_events`).Scan(&actor, &retention, &changes); err != nil {
		t.Fatal(err)
	}
	if actor.Valid || retention != "authentication" || changes != "[]" {
		t.Fatal("anonymous authentication metadata is incorrect")
	}
}

func TestRejectUnsafeFieldsValuesAndMetadata(t *testing.T) {
	db := store(t)
	tx := transaction(t, db)
	cases := map[string]func(*contracts.AuditEvent){
		"missing actor":       func(e *contracts.AuditEvent) { e.ActorID = "" },
		"missing request":     func(e *contracts.AuditEvent) { e.RequestID = "" },
		"missing action":      func(e *contracts.AuditEvent) { e.Action = "" },
		"missing record type": func(e *contracts.AuditEvent) { e.RecordType = "" },
		"missing record id":   func(e *contracts.AuditEvent) { e.RecordID = "" },
		"missing time":        func(e *contracts.AuditEvent) { e.OccurredAt = time.Time{} },
		"unknown auth":        func(e *contracts.AuditEvent) { e.ActorID, e.Action = "", "auth.unknown" },
		"auth prefix":         func(e *contracts.AuditEvent) { e.ActorID, e.Action = "", "auth.login.failed.extra" },
		"record mismatch":     func(e *contracts.AuditEvent) { e.Action = "project.update" },
		"empty project":       func(e *contracts.AuditEvent) { e.ProjectID = ptr("") },
		"request path":        func(e *contracts.AuditEvent) { e.RequestID = "/var/private/db.sqlite" },
		"actor whitespace":    func(e *contracts.AuditEvent) { e.ActorID = " actor-1" },
		"record path":         func(e *contracts.AuditEvent) { e.RecordID = `C:\private\data` },
		"reason error":        func(e *contracts.AuditEvent) { e.Reason = ptr("SQL error at /private/db.sqlite") },
		"reason secret":       func(e *contracts.AuditEvent) { e.Reason = ptr("password=secret") },
		"duplicate":           func(e *contracts.AuditEvent) { e.Changes = append(e.Changes, e.Changes[0]) },
		"both null":           func(e *contracts.AuditEvent) { e.Changes[0].Before, e.Changes[0].After = nil, nil },
		"out of range time":   func(e *contracts.AuditEvent) { e.OccurredAt = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	for _, field := range []string{"password", "password_hash", "token", "session_id", "token_digest", "cookie", "authorization", "api_key", "secret", "internal_path", "error", "notes", "display_name", "Status"} {
		cases["field "+field] = func(e *contracts.AuditEvent) { e.Changes[0].Field = field }
	}
	for _, value := range []string{`{"password":"secret"}`, "/var/db.sqlite", `C:\private\db`, "panic: internal error", "Bearer secret", "active\nsecret", "active "} {
		cases["value "+value] = func(e *contracts.AuditEvent) { e.Changes[0].After = ptr(value) }
	}
	for _, pair := range [][2]string{{"version", "0"}, {"version", "01"}, {"amount_minor", "1.25"}, {"amount_minor", "9223372036854775808"}, {"completion_bps", "10001"}, {"enabled", "yes"}, {"date", "2026-02-30"}, {"role_code", "superuser"}} {
		cases[pair[0]+pair[1]] = func(e *contracts.AuditEvent) {
			e.Changes = []contracts.SafeChange{{Field: pair[0], After: ptr(pair[1])}}
		}
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := event()
			mutate(&e)
			if err := writer.New().Append(context.Background(), tx, e); !errors.Is(err, writer.ErrInvalidEvent) {
				t.Fatalf("got %v, want safe validation error", err)
			}
		})
	}
	var n int
	if err := tx.QueryRow(`SELECT count(*) FROM audit_events`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rejected events wrote rows: %d, %v", n, err)
	}
}

func TestTypedValuesAndSafeFailure(t *testing.T) {
	db := store(t)
	tx := transaction(t, db)
	e := event()
	e.Changes = nil
	for _, pair := range [][2]string{{"version", "2"}, {"amount_minor", "9223372036854775807"}, {"quantity_delta", "-2"}, {"completion_bps", "10000"}, {"enabled", "false"}, {"date", "2026-11-03"}, {"role_code", "administrator"}, {"classification", "restricted"}} {
		e.Changes = append(e.Changes, contracts.SafeChange{Field: pair[0], After: ptr(pair[1])})
	}
	if err := writer.New().Append(context.Background(), tx, e); err != nil {
		t.Fatal(err)
	}
	e.ActorID = "unknown-actor"
	if err := writer.New().Append(context.Background(), tx, e); err != writer.ErrAppend {
		t.Fatalf("foreign key failure must be sanitized: %v", err)
	}
	if err := writer.New().Append(context.Background(), nil, event()); err != writer.ErrInvalidEvent {
		t.Fatal(err)
	}
	var nilTx *sql.Tx
	if err := writer.New().Append(context.Background(), nilTx, event()); err != writer.ErrInvalidEvent {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := writer.New().Append(context.Background(), tx, event()); err != writer.ErrAppend {
		t.Fatalf("closed transaction must fail without fallback: %v", err)
	}
}
