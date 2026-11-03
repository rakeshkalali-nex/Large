package reporting

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/reporting/reports"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	reports.RegisterRoutes(mux, dependencies)
}
