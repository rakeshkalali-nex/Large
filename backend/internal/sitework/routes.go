package sitework

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/sitework/logs"
	"buildestate/backend/internal/sitework/inspections"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	logs.RegisterRoutes(mux, dependencies)
	inspections.RegisterRoutes(mux, dependencies)
}
