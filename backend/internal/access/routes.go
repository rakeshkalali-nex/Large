package access

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/access/auth"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	auth.RegisterRoutes(mux, dependencies)
}
