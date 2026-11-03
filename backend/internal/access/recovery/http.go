package recovery

import (
	"buildestate/backend/internal/access/passwords"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
	"math"
	"net/http"
	"strconv"
)

func RegisterRoutes(mux *http.ServeMux, d contracts.Dependencies) {
	s := New(d)
	mux.HandleFunc("GET /api/v1/auth/recovery-guidance", func(w http.ResponseWriter, r *http.Request) {
		if d.DB == nil {
			transport.Unavailable(w, r)
			return
		}
		g, err := s.Guidance(r.Context())
		if err != nil {
			transactions.WriteError(w, r, err)
			return
		}
		transport.JSON(w, r, 200, g)
	})
	mux.HandleFunc("POST /api/v1/users/{id}/password-reset", func(w http.ResponseWriter, r *http.Request) {
		if !passwords.BrowserMutation(w, r, d) {
			return
		}
		var in Input
		if !transport.DecodeJSON(w, r, &in) {
			return
		}
		out, wait, err := s.Issue(r.Context(), passwords.SessionToken(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), in)
		if wait > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
		}
		if err != nil {
			transactions.WriteError(w, r, err)
			return
		}
		transport.JSON(w, r, 200, out)
	})
}
