package tenant

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/httpx"
)

type createRequest struct {
	Name string `json:"name" validate:"required,max=255"`
}

type updateRequest struct {
	Name *string `json:"name" validate:"omitempty,min=1,max=255"`
}

// RegisterRoutes mounts tenant routes on the router.
func RegisterRoutes(r chi.Router, svc TenantService) {
	r.Route("/tenants", func(r chi.Router) {
		r.Get("/", list(svc))
		r.Post("/", create(svc))
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", get(svc))
			r.Patch("/", update(svc))
			r.Delete("/", deleteTenant(svc))
		})
	})
}

func list(svc TenantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		params, err := httpx.ParseList(r, filterParts(onlyTenant(principal))...)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.List(r.Context(), principal, params.Cursor, params.Limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func create(svc TenantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		var req createRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.Create(r.Context(), principal, CreateInput{Name: req.Name})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func get(svc TenantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.Get(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func update(svc TenantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req updateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.Update(r.Context(), principal, id, UpdateInput{Name: req.Name})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteTenant(svc TenantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.Delete(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func onlyTenant(principal auth.Principal) uuid.UUID {
	if principal.ActorType == auth.ActorSystem {
		return uuid.Nil
	}
	return principal.TenantID
}
