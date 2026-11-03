package finance

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/finance/budgets"
	"buildestate/backend/internal/finance/invoices"
	"buildestate/backend/internal/finance/payments"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	budgets.RegisterRoutes(mux, dependencies)
	invoices.RegisterRoutes(mux, dependencies)
	payments.RegisterRoutes(mux, dependencies)
}
