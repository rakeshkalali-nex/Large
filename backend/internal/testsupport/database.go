// Package testsupport creates disposable real stores; it never reads operator data.
package testsupport

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"buildestate/backend/internal/platform/database"
)

func Database(t testing.TB) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = database.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}
