package budgets

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transport"
)

// RegisterRoutes reserves only unavailable endpoints; it grants no authorization.
func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	mux.HandleFunc("/api/v1/budgets", transport.Unavailable)
	mux.HandleFunc("/api/v1/budgets/{id}", transport.Unavailable)
	mux.HandleFunc("/api/v1/expenses", transport.Unavailable)
	mux.HandleFunc("/api/v1/expenses/{id}", transport.Unavailable)
}
