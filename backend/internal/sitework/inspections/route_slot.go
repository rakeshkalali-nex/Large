package inspections

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transport"
)

// RegisterRoutes reserves only unavailable endpoints; it grants no authorization.
func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	mux.HandleFunc("/api/v1/inspections", transport.Unavailable)
	mux.HandleFunc("/api/v1/inspections/{id}", transport.Unavailable)
	mux.HandleFunc("/api/v1/issues", transport.Unavailable)
	mux.HandleFunc("/api/v1/issues/{id}", transport.Unavailable)
}
