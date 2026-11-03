package settings

import (
	"buildestate/backend/internal/testsupport"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOperatorRealStorageAndDatabaseStatus(t *testing.T) {
	db := testsupport.Database(t)
	dir := t.TempDir()
	d := NewOperatorDiagnostics(db, dir, time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC))
	got := d.Status(context.Background())
	if got.Database.Status != "ready" || got.Attachments.Status != "ready" || got.Backup.Status != "not-configured" || got.Backup.LastSuccessAt != nil || !got.Ready {
		t.Fatalf("dishonest status: %+v", got)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	got = NewOperatorDiagnostics(db, filepath.Join(dir, "missing-private-path"), time.Time{}).Status(context.Background())
	if got.Database.Status != "unavailable" || got.Attachments.Status != "unavailable" || got.Ready {
		t.Fatalf("failure hidden: %+v", got)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), dir) || strings.Contains(string(b), "missing-private-path") || strings.Contains(string(b), "sql:") {
		t.Fatal("diagnostic leaked internals")
	}
}
func TestOperatorAdministratorOnly(t *testing.T) {
	s, ctx := settingsFixture(t)
	overrideSeed(t, s)
	pm := overrideManager(t, s)
	mux := http.NewServeMux()
	s.registerOperator(mux)
	for _, tc := range []struct {
		ctx  context.Context
		want int
	}{{ctx, 200}, {pm, 403}, {context.Background(), 401}} {
		r := httptest.NewRequest("GET", "/api/v1/operator/status", nil).WithContext(tc.ctx)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("got %d want %d: %s", w.Code, tc.want, w.Body.String())
		}
	}
}
