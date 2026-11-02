package voyage

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/httpx"
)

type portCreateRequest struct {
	UnLocode    string   `json:"un_locode" validate:"required,max=5"`
	Name        string   `json:"name" validate:"required,max=255"`
	CountryCode string   `json:"country_code" validate:"required,len=2"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	Timezone    string   `json:"timezone" validate:"omitempty,max=64"`
}

type portUpdateRequest struct {
	Name      *string  `json:"name" validate:"omitempty,min=1,max=255"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Timezone  *string  `json:"timezone" validate:"omitempty,max=64"`
}

type voyageCreateRequest struct {
	ShipID           string     `json:"ship_id" validate:"required,uuid"`
	Code             string     `json:"code" validate:"required,max=64"`
	DepartureAt      time.Time  `json:"departure_at" validate:"required"`
	ArrivalAt        time.Time  `json:"arrival_at" validate:"required"`
	BoardingOpensAt  *time.Time `json:"boarding_opens_at"`
	ManifestFrozenAt *time.Time `json:"manifest_frozen_at"`
}

type voyageUpdateRequest struct {
	Code             *string    `json:"code" validate:"omitempty,min=1,max=64"`
	DepartureAt      *time.Time `json:"departure_at"`
	ArrivalAt        *time.Time `json:"arrival_at"`
	BoardingOpensAt  *time.Time `json:"boarding_opens_at"`
	ManifestFrozenAt *time.Time `json:"manifest_frozen_at"`
}

type portCallCreateRequest struct {
	PortID          string     `json:"port_id" validate:"required,uuid"`
	Sequence        int32      `json:"sequence" validate:"required,min=1"`
	Eta             *time.Time `json:"eta"`
	Etd             *time.Time `json:"etd"`
	AllowsEmbark    *bool      `json:"allows_embark"`
	AllowsDisembark *bool      `json:"allows_disembark"`
}

type portCallUpdateRequest struct {
	Eta             *time.Time `json:"eta"`
	Etd             *time.Time `json:"etd"`
	Ata             *time.Time `json:"ata"`
	Atd             *time.Time `json:"atd"`
	AllowsEmbark    *bool      `json:"allows_embark"`
	AllowsDisembark *bool      `json:"allows_disembark"`
	Status          *string    `json:"status" validate:"omitempty,oneof=scheduled arrived departed skipped"`
}

// RegisterRoutes mounts voyage routes on the router.
func RegisterRoutes(r chi.Router, svc Service) {
	r.Route("/ports", func(r chi.Router) {
		r.Get("/", listPorts(svc))
		r.Post("/", createPort(svc))
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getPort(svc))
			r.Patch("/", updatePort(svc))
			r.Delete("/", deletePort(svc))
		})
	})
	r.Route("/voyages", func(r chi.Router) {
		r.Get("/", listVoyages(svc))
		r.Post("/", createVoyage(svc))
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getVoyage(svc))
			r.Patch("/", updateVoyage(svc))
			r.Delete("/", deleteVoyage(svc))
		})
		r.Route("/{voyageId}/port-calls", func(r chi.Router) {
			r.Get("/", listPortCalls(svc))
			r.Post("/", createPortCall(svc))
		})
	})
	r.Route("/port-calls/{id}", func(r chi.Router) {
		r.Get("/", getPortCall(svc))
		r.Patch("/", updatePortCall(svc))
		r.Delete("/", deletePortCall(svc))
	})
}

func listPorts(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListPorts(r.Context(), principal, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createPort(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		var req portCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreatePort(r.Context(), principal, PortInput{
			UnLocode: req.UnLocode, Name: req.Name, CountryCode: req.CountryCode,
			Latitude: req.Latitude, Longitude: req.Longitude, Timezone: req.Timezone,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getPort(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetPort(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updatePort(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req portUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdatePort(r.Context(), principal, id, PortUpdate{
			Name: req.Name, Latitude: req.Latitude, Longitude: req.Longitude, Timezone: req.Timezone,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deletePort(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeletePort(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listVoyages(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.QueryUUID(r, "ship_id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListVoyages(r.Context(), principal, shipID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createVoyage(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		var req voyageCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := httpx.ParseUUID(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreateVoyage(r.Context(), principal, VoyageInput{
			ShipID: shipID, Code: req.Code, DepartureAt: req.DepartureAt, ArrivalAt: req.ArrivalAt,
			BoardingOpensAt: req.BoardingOpensAt, ManifestFrozenAt: req.ManifestFrozenAt,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getVoyage(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetVoyage(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updateVoyage(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req voyageUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdateVoyage(r.Context(), principal, id, VoyageUpdate{
			Code: req.Code, DepartureAt: req.DepartureAt, ArrivalAt: req.ArrivalAt,
			BoardingOpensAt: req.BoardingOpensAt, ManifestFrozenAt: req.ManifestFrozenAt,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteVoyage(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteVoyage(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listPortCalls(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		voyageID, err := httpx.PathID(r, "voyageId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListPortCalls(r.Context(), principal, voyageID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createPortCall(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		voyageID, err := httpx.PathID(r, "voyageId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req portCallCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		portID, err := httpx.ParseUUID(req.PortID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		embark := true
		if req.AllowsEmbark != nil {
			embark = *req.AllowsEmbark
		}
		disembark := true
		if req.AllowsDisembark != nil {
			disembark = *req.AllowsDisembark
		}
		row, err := svc.CreatePortCall(r.Context(), principal, voyageID, PortCallInput{
			PortID: portID, Sequence: req.Sequence, Eta: req.Eta, Etd: req.Etd,
			AllowsEmbark: embark, AllowsDisembark: disembark,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getPortCall(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetPortCall(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updatePortCall(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req portCallUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdatePortCall(r.Context(), principal, id, PortCallUpdate{
			Eta: req.Eta, Etd: req.Etd, Ata: req.Ata, Atd: req.Atd,
			AllowsEmbark: req.AllowsEmbark, AllowsDisembark: req.AllowsDisembark, Status: req.Status,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deletePortCall(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeletePortCall(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}
