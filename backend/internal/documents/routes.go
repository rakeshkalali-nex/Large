package documents

import (
	"net/http"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/documents/files"
)

func RegisterRoutes(mux *http.ServeMux, dependencies contracts.Dependencies) {
	files.RegisterRoutes(mux, dependencies)
}
