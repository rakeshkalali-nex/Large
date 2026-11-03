package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"buildestate/backend/migrations"
	_ "modernc.org/sqlite"
)

// Open creates a persistent, serialized SQLite connection. DSN pragmas also apply
// to replacement connections, unlike a one-time PRAGMA on the initial connection.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	return openPool(ctx, path, 1, false)
}

// OpenReaders is a bounded query-only pool. Writes must use Open's single writer.
func OpenReaders(ctx context.Context, path string) (*sql.DB, error) {
	return openPool(ctx, path, 4, true)
}

func openPool(ctx context.Context, path string, size int, readOnly bool) (*sql.DB, error) {
	if path == "" || path == ":memory:" {
		return nil, errors.New("persistent database path required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(abs)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	u := url.URL{Scheme: "file", Path: uriPath}
	q := u.Query()
	for _, p := range []string{"foreign_keys(1)", "journal_mode(WAL)", "busy_timeout(5000)", "synchronous(FULL)"} {
		q.Add("_pragma", p)
	}
	q.Set("_txlock", "immediate")
	if readOnly {
		q.Add("_pragma", "query_only(1)")
		q.Set("mode", "ro")
		q.Del("_txlock")
	}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(size)
	db.SetMaxIdleConns(size)
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB) error { return migrateFS(ctx, db, migrations.SQL) }

func migrateFS(ctx context.Context, db *sql.DB, source fs.FS) error {
	names, err := fs.Glob(source, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	if len(names) == 0 {
		return errors.New("migration source is empty")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, checksum TEXT NOT NULL, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT name, checksum FROM schema_migrations ORDER BY name`)
	if err != nil {
		return err
	}
	applied := map[string]string{}
	for rows.Next() {
		var n, c string
		if err = rows.Scan(&n, &c); err != nil {
			rows.Close()
			return err
		}
		applied[n] = c
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	known := map[string]bool{}
	last := ""
	for n := range applied {
		if n > last {
			last = n
		}
	}
	for _, name := range names {
		if !strings.Contains(name, "_") {
			return errors.New("invalid migration name")
		}
		known[name] = true
		body, e := fs.ReadFile(source, name)
		if e != nil {
			return e
		}
		sum := sha256.Sum256(body)
		checksum := hex.EncodeToString(sum[:])
		if old, ok := applied[name]; ok {
			if old != checksum {
				return errors.New("applied migration checksum mismatch")
			}
			continue
		}
		if name < last {
			return errors.New("out-of-order migration")
		}
		if _, err = tx.ExecContext(ctx, string(body)); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations(name,checksum) VALUES(?,?)`, name, checksum); err != nil {
			return err
		}
	}
	for name := range applied {
		if !known[name] {
			return errors.New("applied migration missing from build")
		}
	}
	return tx.Commit()
}
