package contractors

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/contractors/directory"
	"buildestate/backend/internal/contractors/workorders"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	directory.RegisterRoutes(mux, dependencies)
	workorders.RegisterRoutes(mux, dependencies)
}
