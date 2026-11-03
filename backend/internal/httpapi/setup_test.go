package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/database"
	"buildestate/backend/internal/platform/transport"
)

func TestSetupRouterHealthStaticAndAPIBoundary(t *testing.T) {
	dir := t.TempDir()
	static := filepath.Join(dir, "static")
	attachments := filepath.Join(dir, "attachments")
	for _, d := range []string{static, attachments} {
		if err := os.Mkdir(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(static, "index.html"), []byte("<!doctype html><title>BuildEstate</title>"), 0600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	handler := New(contracts.Dependencies{DB: db}, static, attachments, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, tc := range []struct {
		path   string
		status int
		json   bool
	}{
		{"/api/v1/health/live", 200, true}, {"/api/v1/health/ready", 200, true},
		{"/api/v1/__setup_unknown_route__", 404, true}, {"/api", 404, true},
		{"/projects/setup-record/plan", 200, false}, {"/missing.js", 404, false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if tc.json {
				var body map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["requestId"] == "" {
					t.Fatal("missing request ID")
				}
			}
		})
	}
	db.Close()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/health/ready", nil))
	if w.Code != 503 {
		t.Fatal("closed database reported ready")
	}
}

func TestSetupStrictJSONAndPanicRedaction(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name string `json:"name"`
		}
		if transport.DecodeJSON(w, r, &input) {
			transport.JSON(w, r, 200, input)
		}
	}), logger)
	for _, tc := range []struct {
		body   string
		status int
	}{{`{"name":"valid"}`, 200}, {`{"unexpected":true}`, 400}, {`{} {}`, 400}, {strings.Repeat("x", int(transport.MaxJSONBody)+1), 413}} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/v1/setup-test", strings.NewReader(tc.body))
		r.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("got %d expected %d", w.Code, tc.status)
		}
	}
	w := httptest.NewRecorder()
	Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("private database path") }), logger).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 500 || strings.Contains(w.Body.String(), "private") {
		t.Fatal("panic was not sanitized")
	}
}
