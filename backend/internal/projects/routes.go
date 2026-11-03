package projects

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/projects/directory"
	"buildestate/backend/internal/projects/plan"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	directory.RegisterRoutes(mux, dependencies)
	plan.RegisterRoutes(mux, dependencies)
}
