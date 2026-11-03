package config

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Root, Mode, Port, DataDir, SQLitePath, AttachmentsDir, BackupDir, StaticDir string
	CookieSecure bool
}

// Load reads an optional root .env; process environment always takes precedence.
func Load() (Config, error) {
	bad := errors.New("invalid application configuration")
	root := os.Getenv("APP_ROOT")
	if root == "" { root = "." }
	root, err := filepath.Abs(root)
	if err != nil { return Config{}, bad }
	values := map[string]string{}
	f, err := os.Open(filepath.Join(root, ".env"))
	if err != nil && !errors.Is(err, os.ErrNotExist) { return Config{}, bad }
	if err == nil {
		defer f.Close()
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line == "" || strings.HasPrefix(line, "#") { continue }
			key, value, ok := strings.Cut(line, "=")
			if !ok { return Config{}, bad }
			key, value = strings.TrimSpace(key), strings.TrimSpace(value)
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) { value = value[1:len(value)-1] }
			values[key] = value
		}
		if s.Err() != nil { return Config{}, bad }
	}
	get := func(key, fallback string) string { if v, ok := os.LookupEnv(key); ok { return v }; if v, ok := values[key]; ok { return v }; return fallback }
	resolve := func(v string) string { if !filepath.IsAbs(v) { v = filepath.Join(root, v) }; return filepath.Clean(v) }
	c := Config{Root:root, Mode:get("APP_MODE", "development"), Port:get("PORT", "8080"), DataDir:resolve(get("DATA_DIR", "./var/local")), SQLitePath:resolve(get("SQLITE_PATH", "./var/local/buildestate.sqlite")), AttachmentsDir:resolve(get("ATTACHMENTS_DIR", "./var/local/attachments")), BackupDir:resolve(get("BACKUP_DIR", "./var/backups")), StaticDir:resolve(get("STATIC_DIR", "./frontend/dist/buildestate/browser"))}
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 0 || port > 65535 { return Config{}, bad }
	c.CookieSecure, err = strconv.ParseBool(get("COOKIE_SECURE", "false"))
	if err != nil || (c.Mode != "development" && c.Mode != "production" && c.Mode != "test") { return Config{}, bad }
	// Never expose private data through the static root; never co-locate backups.
	overlap := func(a,b string) bool { a, b = strings.ToLower(a), strings.ToLower(b); return a == b || strings.HasPrefix(a,b+string(filepath.Separator)) || strings.HasPrefix(b,a+string(filepath.Separator)) }
	if overlap(c.StaticDir,c.DataDir) || overlap(c.StaticDir,c.AttachmentsDir) || overlap(c.StaticDir,c.BackupDir) || overlap(c.StaticDir,c.SQLitePath) || overlap(c.BackupDir,c.DataDir) || overlap(c.BackupDir,c.AttachmentsDir) || overlap(c.AttachmentsDir,c.SQLitePath) { return Config{}, bad }
	for _, dir := range []string{c.DataDir,c.AttachmentsDir,c.BackupDir,c.StaticDir} { if st,e := os.Stat(dir); e == nil && !st.IsDir() { return Config{},bad } else if e != nil && !errors.Is(e,os.ErrNotExist) { return Config{},bad } }
	if st,e := os.Stat(c.SQLitePath); e == nil && st.IsDir() { return Config{},bad }
	return c,nil
}
