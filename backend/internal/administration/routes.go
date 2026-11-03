package administration

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/administration/users"
	"buildestate/backend/internal/administration/settings"
	"buildestate/backend/internal/administration/audit"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	users.RegisterRoutes(mux, dependencies)
	settings.RegisterRoutes(mux, dependencies)
	audit.RegisterRoutes(mux, dependencies)
}
