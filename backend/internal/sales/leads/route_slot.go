package leads

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transport"
)

// RegisterRoutes reserves only unavailable endpoints; it grants no authorization.
func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	mux.HandleFunc("/api/v1/leads", transport.Unavailable)
	mux.HandleFunc("/api/v1/leads/{id}", transport.Unavailable)
	mux.HandleFunc("/api/v1/customers", transport.Unavailable)
	mux.HandleFunc("/api/v1/customers/{id}", transport.Unavailable)
	mux.HandleFunc("/api/v1/site-visits", transport.Unavailable)
	mux.HandleFunc("/api/v1/site-visits/{id}", transport.Unavailable)
}
