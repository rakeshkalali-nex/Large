package sales

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/sales/properties"
	"buildestate/backend/internal/sales/leads"
	"buildestate/backend/internal/sales/bookings"
	"buildestate/backend/internal/sales/sale"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	properties.RegisterRoutes(mux, dependencies)
	leads.RegisterRoutes(mux, dependencies)
	bookings.RegisterRoutes(mux, dependencies)
	sale.RegisterRoutes(mux, dependencies)
}
