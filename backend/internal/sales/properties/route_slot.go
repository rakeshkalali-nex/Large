package properties

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transport"
)

// RegisterRoutes reserves only unavailable endpoints; it grants no authorization.
func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	mux.HandleFunc("/api/v1/properties", transport.Unavailable)
	mux.HandleFunc("/api/v1/properties/{id}", transport.Unavailable)
	mux.HandleFunc("/api/v1/reservations", transport.Unavailable)
	mux.HandleFunc("/api/v1/reservations/{id}", transport.Unavailable)
}
