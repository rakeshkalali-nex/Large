package settings

import (
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/storage"
	"context"
	"database/sql"
	"net/http"
	"runtime/debug"
	"time"
)

type localOperatorDiagnostics struct {
	db             *sql.DB
	attachmentsDir string
	startedAt      time.Time
}

// NewOperatorDiagnostics does not create storage or claim a backup has run.
// Wire once at process startup with the configured attachment root.
func NewOperatorDiagnostics(db *sql.DB, attachmentsDir string, startedAt time.Time) contracts.OperatorDiagnostics {
	return localOperatorDiagnostics{db: db, attachmentsDir: attachmentsDir, startedAt: startedAt}
}
func (d localOperatorDiagnostics) Status(ctx context.Context) contracts.OperatorStatus {
	out := contracts.OperatorStatus{Version: "unknown", Database: contracts.OperatorComponent{Status: "unavailable"}, Attachments: contracts.OperatorComponent{Status: "not-configured"}, Backup: contracts.OperatorBackup{Status: "not-configured"}}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		out.Version = info.Main.Version
	}
	if !d.startedAt.IsZero() {
		out.StartedAt = d.startedAt.UTC().Format(time.RFC3339Nano)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if d.db != nil && d.db.PingContext(ctx) == nil {
		out.Database.Status = "ready"
	} else {
		id := "DATABASE_UNAVAILABLE"
		out.Database.DiagnosticID = &id
	}
	if d.attachmentsDir != "" {
		out.Attachments.Status = "unavailable"
		if storage.Check(d.attachmentsDir) == nil {
			out.Attachments.Status = "ready"
		}
		if out.Attachments.Status != "ready" {
			id := "ATTACHMENTS_UNAVAILABLE"
			out.Attachments.DiagnosticID = &id
		}
	}
	// The HTTP server is composed after startup checks succeed. Both readiness
	// consumers use this live diagnostic result; backup protection is separate.
	out.Ready = out.Database.Status == "ready" && out.Attachments.Status == "ready"
	return out
}
func (s Service) registerOperator(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/operator/status", s.handle("operator.read", func(w http.ResponseWriter, r *http.Request) {
		provider := s.D.Operator
		if provider == nil {
			provider = NewOperatorDiagnostics(s.D.DB, "", time.Time{})
		}
		respond(w, r, provider.Status(r.Context()), nil)
	}))
}
