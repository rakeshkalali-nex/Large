package users

import (
	accessusers "buildestate/backend/internal/access/users"
	"buildestate/backend/internal/contracts"
	"net/http"
)

// RegisterRoutes reserves only unavailable endpoints; it grants no authorization.
func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	accessusers.RegisterRoutes(mux, dependencies)
}
