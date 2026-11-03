package settings_test

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/httpapi"
	"buildestate/backend/internal/settings"
	"buildestate/backend/internal/testsupport"
)

func TestOperatorAndHealthReadinessAgree(t *testing.T) {
	for _, state := range []string{"healthy", "database-closed", "database-missing", "attachments-missing", "attachments-unconfigured"} {
		t.Run(state, func(t *testing.T) {
			db := testsupport.Database(t)
			dir := t.TempDir()
			switch state {
			case "database-closed":
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
			case "database-missing":
				db = nil
			case "attachments-missing":
				dir = filepath.Join(dir, "private-missing-path")
			case "attachments-unconfigured":
				dir = ""
			}
			status := settings.NewOperatorDiagnostics(db, dir, time.Now()).Status(context.Background())
			wantReady := state == "healthy"
			if status.Ready != wantReady {
				t.Fatalf("ready = %v, want %v", status.Ready, wantReady)
			}
			handler := httpapi.New(contracts.Dependencies{DB: db}, t.TempDir(), dir, slog.New(slog.NewTextHandler(io.Discard, nil)))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/health/ready", nil))
			wantCode := 503
			if wantReady {
				wantCode = 200
			}
			if w.Code != wantCode {
				t.Fatalf("health = %d, want %d: %s", w.Code, wantCode, w.Body.String())
			}
			if status.Database.Status == "unavailable" && status.Database.DiagnosticID == nil {
				t.Fatal("missing database diagnostic")
			}
			if status.Attachments.Status == "unavailable" && status.Attachments.DiagnosticID == nil {
				t.Fatal("missing attachment diagnostic")
			}
		})
	}
}

type startupPendingDiagnostics struct{}

func (startupPendingDiagnostics) Status(context.Context) contracts.OperatorStatus {
	return contracts.OperatorStatus{Ready: false, Database: contracts.OperatorComponent{Status: "ready"}, Attachments: contracts.OperatorComponent{Status: "ready"}}
}

func TestHealthHonorsOperatorStartupReadiness(t *testing.T) {
	// A provider can withhold readiness for startup work even with healthy storage.
	handler := httpapi.New(contracts.Dependencies{DB: testsupport.Database(t), Operator: startupPendingDiagnostics{}}, t.TempDir(), t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/health/ready", nil))
	if w.Code != 503 {
		t.Fatalf("startup pending health = %d, want 503", w.Code)
	}
}
