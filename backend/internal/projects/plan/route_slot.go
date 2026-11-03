package plan

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transport"
)

// RegisterRoutes reserves only unavailable endpoints; it grants no authorization.
func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	mux.HandleFunc("/api/v1/projects/{id}/plan", transport.Unavailable)
	mux.HandleFunc("/api/v1/projects/{id}/milestones", transport.Unavailable)
	mux.HandleFunc("/api/v1/projects/{id}/tasks", transport.Unavailable)
}
