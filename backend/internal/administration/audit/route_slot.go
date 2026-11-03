package audit

import (
	"buildestate/backend/internal/audit/history"
	"buildestate/backend/internal/contracts"
	"net/http"
)

// RegisterRoutes reserves only unavailable endpoints; it grants no authorization.
func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	history.RegisterRoutes(mux, dependencies)
}
