package posting

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"buildestate/backend/internal/testsupport"
)

func readyDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db := testsupport.Database(t)
	_, err := db.Exec(`INSERT INTO users(id,display_name,email,normalized_email,role_code) VALUES('operator','Operator','test@example.invalid','test@example.invalid','administrator');
	INSERT INTO company_settings(id,company_name,wordmark,currency_code,currency_minor_digits,timezone,timezone_confirmed,recovery_instructions,recovery_contact_label,notice_version,retention_schedule_version,operational_policy_confirmed_at,created_by)
	VALUES('company','Test','Test','USD',2,'Asia/Kolkata',1,'Operator procedure','Operator','notice-1','retention-1','2026-11-03T00:00:00Z','operator');
	INSERT INTO operating_countries(settings_id,country_code) VALUES('company','IN');`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRequireReady(t *testing.T) {
	ctx := context.Background()
	t.Run("missing company", func(t *testing.T) {
		db := testsupport.Database(t)
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err = (Configuration{}).RequireReady(ctx, tx); !errors.Is(err, ErrNotReady) {
			t.Fatalf("%v", err)
		}
	})
	t.Run("ready", func(t *testing.T) {
		db := readyDatabase(t)
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		got, err := (Configuration{}).RequireReady(ctx, tx)
		if err != nil || got.CurrencyCode != "USD" || got.MinorDigits != 2 || got.Timezone != "Asia/Kolkata" {
			t.Fatalf("%+v %v", got, err)
		}
		var locked int
		if err = tx.QueryRow(`SELECT currency_locked FROM company_settings`).Scan(&locked); err != nil || locked != 0 {
			t.Fatalf("read changed lock: %d %v", locked, err)
		}
	})
	for name, statement := range map[string]string{
		"currency absent":          `UPDATE company_settings SET currency_code=NULL,currency_minor_digits=NULL`,
		"unknown currency":         `UPDATE company_settings SET currency_code='ZZZ'`,
		"test currency":            `UPDATE company_settings SET currency_code='XTS'`,
		"timezone unconfirmed":     `UPDATE company_settings SET timezone_confirmed=0`,
		"timezone invalid":         `UPDATE company_settings SET timezone='Invalid/Zone'`,
		"machine timezone":         `UPDATE company_settings SET timezone='Local'`,
		"notice absent":            `UPDATE company_settings SET notice_version=NULL`,
		"retention blank":          `UPDATE company_settings SET retention_schedule_version=' '`,
		"policy unconfirmed":       `UPDATE company_settings SET operational_policy_confirmed_at=NULL`,
		"policy timestamp invalid": `UPDATE company_settings SET operational_policy_confirmed_at='yesterday'`,
		"countries absent":         `DELETE FROM operating_countries`,
		"country malformed":        `UPDATE operating_countries SET country_code='12'`,
	} {
		t.Run(name, func(t *testing.T) {
			db := readyDatabase(t)
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if _, err = tx.Exec(statement); err != nil {
				t.Fatal(err)
			}
			if _, err = (Configuration{}).RequireReady(ctx, tx); !errors.Is(err, ErrNotReady) {
				t.Fatalf("%v", err)
			}
			if err = (Configuration{}).LockCurrency(ctx, tx); !errors.Is(err, ErrNotReady) {
				t.Fatalf("lock: %v", err)
			}
			var locked int
			if err = tx.QueryRow(`SELECT currency_locked FROM company_settings`).Scan(&locked); err != nil || locked != 0 {
				t.Fatalf("invalid setup locked: %d %v", locked, err)
			}
		})
	}
}

func TestTransactionalCurrencyLock(t *testing.T) {
	ctx := context.Background()
	db := readyDatabase(t)
	c := Configuration{}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.LockCurrency(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE company_settings SET company_name='rolled back'`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var locked int
	var name string
	if err = db.QueryRow(`SELECT currency_locked,company_name FROM company_settings`).Scan(&locked, &name); err != nil || locked != 0 || name != "Test" {
		t.Fatalf("rollback: %d %s %v", locked, name, err)
	}
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = c.LockCurrency(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if err = c.LockCurrency(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT currency_locked FROM company_settings`).Scan(&locked); err != nil || locked != 1 {
		t.Fatalf("commit: %d %v", locked, err)
	}
	for _, statement := range []string{`UPDATE company_settings SET currency_code='INR'`, `UPDATE company_settings SET currency_minor_digits=3`, `UPDATE company_settings SET currency_locked=0`} {
		if _, err = db.Exec(statement); err == nil {
			t.Fatalf("currency guard accepted %s", statement)
		}
	}
}
