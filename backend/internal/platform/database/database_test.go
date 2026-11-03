package database

import (
	"context"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestMigrationChecksumsAndConnectionPragmas(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "checks.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	source := fstest.MapFS{"0001_test.sql": &fstest.MapFile{Data: []byte("CREATE TABLE example(id INTEGER PRIMARY KEY) STRICT;")}}
	if err = migrateFS(ctx, db, source); err != nil {
		t.Fatal(err)
	}
	if err = migrateFS(ctx, db, source); err != nil {
		t.Fatal(err)
	}
	source["0001_test.sql"].Data = []byte("CREATE TABLE changed(id INTEGER PRIMARY KEY) STRICT;")
	if migrateFS(ctx, db, source) == nil {
		t.Fatal("changed applied migration accepted")
	}
	db.SetMaxIdleConns(0)
	for key, want := range map[string]int{"foreign_keys": 1, "busy_timeout": 5000, "synchronous": 2} {
		var got int
		if err = db.QueryRowContext(ctx, "PRAGMA "+key).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s: %d", key, got)
		}
	}
	var mode string
	if err = db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("journal mode %q: %v", mode, err)
	}
}
