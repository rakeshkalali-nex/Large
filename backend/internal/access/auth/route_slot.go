package auth

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transport"
)

// RegisterRoutes reserves only unavailable endpoints; it grants no authorization.
func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	mux.HandleFunc("/api/v1/auth/csrf", transport.Unavailable)
	mux.HandleFunc("/api/v1/auth/login", transport.Unavailable)
	mux.HandleFunc("/api/v1/auth/me", transport.Unavailable)
	mux.HandleFunc("/api/v1/auth/logout", transport.Unavailable)
	mux.HandleFunc("/api/v1/auth/change-password", transport.Unavailable)
	mux.HandleFunc("/api/v1/auth/reset-password", transport.Unavailable)
	mux.HandleFunc("/api/v1/auth/recovery-guidance", transport.Unavailable)
}
