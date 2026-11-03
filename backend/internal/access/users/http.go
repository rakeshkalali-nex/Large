package users

import (
	"buildestate/backend/internal/access/passwords"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, d contracts.Dependencies) {
	s := New(d)
	respond := func(w http.ResponseWriter, r *http.Request, data any, err error) {
		if err != nil {
			transactions.WriteError(w, r, err)
			return
		}
		transport.JSON(w, r, 200, data)
	}
	mux.HandleFunc("GET /api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		items, e := s.List(r.Context(), passwords.SessionToken(r), r.URL.Query().Get("search"))
		respond(w, r, map[string]any{"items": items}, e)
	})
	for path, projects := range map[string]bool{"/api/v1/roles": false, "/api/v1/users/projects": true} {
		mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
			items, e := s.Options(r.Context(), passwords.SessionToken(r), projects)
			respond(w, r, map[string]any{"items": items}, e)
		})
	}
	for _, method := range []string{"POST /api/v1/users", "PUT /api/v1/users/{id}"} {
		mux.HandleFunc(method, func(w http.ResponseWriter, r *http.Request) {
			if !passwords.BrowserMutation(w, r, d) {
				return
			}
			var in Input
			if !transport.DecodeJSON(w, r, &in) {
				return
			}
			out, e := s.Save(r.Context(), passwords.SessionToken(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), in)
			respond(w, r, out, e)
		})
	}
	for _, action := range []string{"deactivate", "reactivate"} {
		mux.HandleFunc("POST /api/v1/users/{id}/"+action, func(w http.ResponseWriter, r *http.Request) {
			if !passwords.BrowserMutation(w, r, d) {
				return
			}
			var in Transition
			if !transport.DecodeJSON(w, r, &in) {
				return
			}
			out, e := s.ChangeStatus(r.Context(), passwords.SessionToken(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), action, in)
			respond(w, r, out, e)
		})
	}
}
