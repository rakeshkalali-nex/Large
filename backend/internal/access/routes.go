package access

import (
	"buildestate/backend/internal/access/auth"
	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/contracts"
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) *sessions.Handler {
	auth.RegisterRoutes(mux, dependencies)
	h := sessions.NewHandler(dependencies)
	h.RegisterRoutes(mux)
	return h
}
