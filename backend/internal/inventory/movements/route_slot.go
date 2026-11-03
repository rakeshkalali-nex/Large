package movements

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transport"
)

// RegisterRoutes reserves only unavailable endpoints; it grants no authorization.
func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	mux.HandleFunc("/api/v1/stock-movements", transport.Unavailable)
	mux.HandleFunc("/api/v1/stock/issues", transport.Unavailable)
	mux.HandleFunc("/api/v1/stock/returns", transport.Unavailable)
	mux.HandleFunc("/api/v1/stock/transfers", transport.Unavailable)
	mux.HandleFunc("/api/v1/stock/adjustments", transport.Unavailable)
}
