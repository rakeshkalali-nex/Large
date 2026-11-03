package inventory

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/inventory/balances"
	"buildestate/backend/internal/inventory/movements"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	balances.RegisterRoutes(mux, dependencies)
	movements.RegisterRoutes(mux, dependencies)
}
