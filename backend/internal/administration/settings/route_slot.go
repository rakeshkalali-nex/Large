package settings

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transport"
)

// RegisterRoutes reserves only unavailable endpoints; it grants no authorization.
func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	mux.HandleFunc("/api/v1/settings", transport.Unavailable)
	mux.HandleFunc("/api/v1/operator/status", transport.Unavailable)
	mux.HandleFunc("/api/v1/policy-acknowledgments", transport.Unavailable)
	mux.HandleFunc("/api/v1/policy-acknowledgments/{id}", transport.Unavailable)
	mux.HandleFunc("/api/v1/projects/{id}/overrides", transport.Unavailable)
}
