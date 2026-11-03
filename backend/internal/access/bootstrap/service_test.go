package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/platform/database"
	"buildestate/backend/internal/testsupport"
)

func validInput() Input {
	return Input{DisplayName: "Local Operator", Email: " Operator@Example.invalid ", Password: "private bootstrap phrase", CompanyName: "Operator Company", RecoveryContactLabel: "Company help desk", RecoveryInstructions: "Contact the help desk in person for identity verification."}
}

func counts(t *testing.T, db *sql.DB, users, settings, audits int) {
	t.Helper()
	for table, want := range map[string]int{"users": users, "company_settings": settings, "audit_events": audits} {
		var got int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil || got != want {
			t.Fatalf("%s count = %d; want %d; error %v", table, got, want, err)
		}
	}
}

func TestCreatePersistsHashRecoveryAndSafeAudit(t *testing.T) {
	db := testsupport.Database(t)
	in := validInput()
	if err := Create(context.Background(), db, in); err != nil {
		t.Fatal(err)
	}
	counts(t, db, 1, 1, 2)
	var email, role, status, hash string
	var setup int
	if err := db.QueryRow(`SELECT normalized_email,role_code,status,password_hash,setup_required FROM users`).Scan(&email, &role, &status, &hash, &setup); err != nil {
		t.Fatal(err)
	}
	if email != "operator@example.invalid" || role != "administrator" || status != "active" || setup != 0 || !sessions.VerifyPassword(hash, in.Password) {
		t.Fatal("invalid initialized account")
	}
	var contact, instructions string
	if err := db.QueryRow(`SELECT recovery_contact_label,recovery_instructions FROM company_settings WHERE id='company'`).Scan(&contact, &instructions); err != nil {
		t.Fatal(err)
	}
	if contact != in.RecoveryContactLabel || instructions != in.RecoveryInstructions {
		t.Fatal("guidance not persisted")
	}
	var leaks int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE instr(changes_json,?)>0 OR instr(COALESCE(reason,''),?)>0`, in.Password, in.Password).Scan(&leaks); err != nil || leaks != 0 {
		t.Fatal("credential leaked into audit")
	}
	if err := Create(context.Background(), db, in); !errors.Is(err, ErrExists) {
		t.Fatalf("repeat: %v", err)
	}
	counts(t, db, 1, 1, 2)
}

func TestAnyExistingStaffBlocksBootstrap(t *testing.T) {
	for _, status := range []string{"setup-required", "active", "deactivated"} {
		t.Run(status, func(t *testing.T) {
			db := testsupport.Database(t)
			if _, err := db.Exec(`INSERT INTO users(id,display_name,email,normalized_email,role_code,status) VALUES('existing','Existing','existing@example.invalid','existing@example.invalid','site-engineer',?)`, status); err != nil {
				t.Fatal(err)
			}
			if err := Create(context.Background(), db, validInput()); !errors.Is(err, ErrExists) {
				t.Fatalf("existing staff: %v", err)
			}
			counts(t, db, 1, 0, 0)
		})
	}
}

func TestValidationAndUnicodePassword(t *testing.T) {
	for _, password := range []string{"short", strings.Repeat("a", 129), string([]byte{0xff})} {
		db := testsupport.Database(t)
		in := validInput()
		in.Password = password
		if err := Create(context.Background(), db, in); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid password accepted")
		}
		counts(t, db, 0, 0, 0)
	}
	db := testsupport.Database(t)
	in := validInput()
	in.Password = strings.Repeat("界", 12)
	if err := Create(context.Background(), db, in); err != nil {
		t.Fatal(err)
	}
}

func TestAuditFailureRollsBackEverything(t *testing.T) {
	db := testsupport.Database(t)
	// Fail the second audit insert, after account, settings and first event writes.
	if _, err := db.Exec(`CREATE TRIGGER bootstrap_audit_fault BEFORE INSERT ON audit_events WHEN NEW.record_type='settings' BEGIN SELECT RAISE(ABORT,'private driver diagnostic'); END`); err != nil {
		t.Fatal(err)
	}
	if err := Create(context.Background(), db, validInput()); !errors.Is(err, ErrFailed) {
		t.Fatalf("unsafe error: %v", err)
	}
	counts(t, db, 0, 0, 0)
}

func TestConcurrentIndependentConnectionsAndRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "bootstrap.sqlite")
	first, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if err = database.Migrate(ctx, first); err != nil {
		t.Fatal(err)
	}
	second, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, db := range []*sql.DB{first, second} {
		go func(db *sql.DB) { <-start; results <- Create(ctx, db, validInput()) }(db)
	}
	close(start)
	ok, rejected := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			ok++
		} else if errors.Is(err, ErrExists) {
			rejected++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || rejected != 1 {
		t.Fatal("bootstrap race was not serialized")
	}
	counts(t, first, 1, 1, 2)
	first.Close()
	second.Close()
	reopened, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if err = Create(ctx, reopened, validInput()); !errors.Is(err, ErrExists) {
		t.Fatalf("restart: %v", err)
	}
	counts(t, reopened, 1, 1, 2)
}
