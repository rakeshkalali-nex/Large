package administration

import (
	"buildestate/backend/internal/administration/audit"
	"buildestate/backend/internal/administration/users"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/settings"
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	users.RegisterRoutes(mux, dependencies)
	settings.RegisterRoutes(mux, dependencies)
	audit.RegisterRoutes(mux, dependencies)
}
