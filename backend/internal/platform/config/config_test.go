package config

import (
	"path/filepath"
	"testing"
)

func TestPathsResolveFromRootAndRejectPrivateStaticOverlap(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APP_ROOT", root)
	t.Setenv("APP_MODE", "test")
	t.Setenv("PORT", "0")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("DATA_DIR", "data")
	t.Setenv("SQLITE_PATH", "data/db.sqlite")
	t.Setenv("ATTACHMENTS_DIR", "data/files")
	t.Setenv("BACKUP_DIR", "backups")
	t.Setenv("STATIC_DIR", "public")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.SQLitePath != filepath.Join(root, "data/db.sqlite") {
		t.Fatal("database was not resolved from app root")
	}
	t.Setenv("STATIC_DIR", "data")
	if _, err = Load(); err == nil {
		t.Fatal("private data exposed through static directory")
	}
	t.Setenv("STATIC_DIR", "public")
	t.Setenv("PORT", "65536")
	if _, err = Load(); err == nil {
		t.Fatal("invalid port accepted")
	}
}
