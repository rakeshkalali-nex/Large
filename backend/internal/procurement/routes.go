package procurement

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/procurement/catalog"
	"buildestate/backend/internal/procurement/orders"
	"buildestate/backend/internal/procurement/receipts"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	catalog.RegisterRoutes(mux, dependencies)
	orders.RegisterRoutes(mux, dependencies)
	receipts.RegisterRoutes(mux, dependencies)
}
