package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"buildestate/backend/internal/access"
	"buildestate/backend/internal/administration"
	"buildestate/backend/internal/contractors"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/documents"
	"buildestate/backend/internal/finance"
	"buildestate/backend/internal/inventory"
	"buildestate/backend/internal/platform/storage"
	"buildestate/backend/internal/platform/transport"
	"buildestate/backend/internal/procurement"
	"buildestate/backend/internal/projects"
	"buildestate/backend/internal/reporting"
	"buildestate/backend/internal/sales"
	"buildestate/backend/internal/sitework"
)

// New explicitly composes feature modules. Nil services never grant permissions.
func New(d contracts.Dependencies, staticDir, attachmentsDir string, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	authentication := access.RegisterRoutes(mux, d)
	administration.RegisterRoutes(mux, d)
	contractors.RegisterRoutes(mux, d)
	documents.RegisterRoutes(mux, d)
	finance.RegisterRoutes(mux, d)
	inventory.RegisterRoutes(mux, d)
	procurement.RegisterRoutes(mux, d)
	projects.RegisterRoutes(mux, d)
	reporting.RegisterRoutes(mux, d)
	sales.RegisterRoutes(mux, d)
	sitework.RegisterRoutes(mux, d)
	mux.HandleFunc("GET /api/v1/health/live", func(w http.ResponseWriter, r *http.Request) {
		transport.JSON(w, r, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if d.DB == nil || d.DB.PingContext(ctx) != nil || storage.Check(attachmentsDir) != nil {
			transport.Error(w, r, 503, "SERVICE_UNAVAILABLE", "Service is not ready.")
			return
		}
		transport.JSON(w, r, 200, map[string]string{"status": "ready"})
	})
	mux.Handle("/", staticHandler(staticDir))
	// ServeMux's built-in 405 uses plain text; API failures must retain the envelope.
	protected := authentication.Protect(mux)
	dispatch := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)
		if pattern == "" {
			transport.Error(w, r, 405, "METHOD_NOT_ALLOWED", "Method not allowed.")
			return
		}
		if pattern == "/" {
			mux.ServeHTTP(w, r)
			return
		}
		protected.ServeHTTP(w, r)
	})
	return Middleware(dispatch, logger)
}
