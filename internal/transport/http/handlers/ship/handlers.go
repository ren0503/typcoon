package ship

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ren0503/tycoon/internal/application/auth"
	appship "github.com/ren0503/tycoon/internal/application/ship"
	"github.com/ren0503/tycoon/internal/transport/http/httpx"
)

type shipCreateRequest struct {
	Name      string  `json:"name" validate:"required,max=255"`
	ImoNumber *string `json:"imo_number" validate:"omitempty,max=32"`
	FlagState *string `json:"flag_state" validate:"omitempty,max=64"`
	Timezone  string  `json:"timezone" validate:"omitempty,max=64"`
	Status    string  `json:"status" validate:"omitempty,oneof=active maintenance retired"`
}

type shipUpdateRequest struct {
	Name      *string `json:"name" validate:"omitempty,min=1,max=255"`
	ImoNumber *string `json:"imo_number" validate:"omitempty,max=32"`
	FlagState *string `json:"flag_state" validate:"omitempty,max=64"`
	Timezone  *string `json:"timezone" validate:"omitempty,max=64"`
	Status    *string `json:"status" validate:"omitempty,oneof=active maintenance retired"`
}

type deckCreateRequest struct {
	Level *int32 `json:"level" validate:"required,min=0"`
	Name  string `json:"name" validate:"required,max=255"`
}

type deckUpdateRequest struct {
	Level *int32  `json:"level" validate:"omitempty,min=0"`
	Name  *string `json:"name" validate:"omitempty,min=1,max=255"`
}

type areaCreateRequest struct {
	DeckID   string          `json:"deck_id" validate:"required,uuid"`
	Name     string          `json:"name" validate:"required,max=255"`
	AreaType string          `json:"area_type" validate:"required,oneof=restaurant lobby open_deck theatre shop corridor stairwell spa pool other"`
	Capacity *int32          `json:"capacity" validate:"omitempty,min=0"`
	Layout   json.RawMessage `json:"layout"`
}

type areaUpdateRequest struct {
	Name     *string         `json:"name" validate:"omitempty,min=1,max=255"`
	Capacity *int32          `json:"capacity" validate:"omitempty,min=0"`
	Layout   json.RawMessage `json:"layout"`
}

type roomCreateRequest struct {
	DeckID     string          `json:"deck_id" validate:"required,uuid"`
	Number     string          `json:"number" validate:"required,max=32"`
	RoomType   string          `json:"room_type" validate:"required,oneof=inside oceanview balcony suite crew other"`
	Capacity   *int16          `json:"capacity" validate:"required,min=1"`
	IsBookable *bool           `json:"is_bookable"`
	Layout     json.RawMessage `json:"layout"`
}

type roomUpdateRequest struct {
	DeckID     *string         `json:"deck_id" validate:"omitempty,uuid"`
	Number     *string         `json:"number" validate:"omitempty,min=1,max=32"`
	RoomType   *string         `json:"room_type" validate:"omitempty,oneof=inside oceanview balcony suite crew other"`
	Capacity   *int16          `json:"capacity" validate:"omitempty,min=1"`
	IsBookable *bool           `json:"is_bookable"`
	Layout     json.RawMessage `json:"layout"`
}

// RegisterRoutes mounts ship routes on the router.
func RegisterRoutes(r chi.Router, svc appship.Service) {
	r.Route("/ships", func(r chi.Router) {
		r.Get("/", listShips(svc))
		r.Post("/", createShip(svc))
		r.Get("/structure/import-template", structureImportTemplate(svc))
		r.Route("/{shipId}/structure/imports", func(r chi.Router) {
			r.Post("/", uploadStructureImport(svc))
			r.Route("/{importId}", func(r chi.Router) {
				r.Get("/", getStructureImport(svc))
				r.Get("/preview", previewStructureImport(svc))
				r.Post("/confirm", confirmStructureImport(svc))
			})
		})
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getShip(svc))
			r.Patch("/", updateShip(svc))
			r.Delete("/", deleteShip(svc))
		})
		r.Route("/{shipId}/decks", func(r chi.Router) {
			r.Get("/", listDecks(svc))
			r.Post("/", createDeck(svc))
		})
		r.Route("/{shipId}/areas", func(r chi.Router) {
			r.Get("/", listAreas(svc))
			r.Post("/", createArea(svc))
		})
		r.Route("/{shipId}/rooms", func(r chi.Router) {
			r.Get("/", listRooms(svc))
			r.Post("/", createRoom(svc))
		})
	})
	r.Route("/decks/{id}", func(r chi.Router) {
		r.Get("/", getDeck(svc))
		r.Patch("/", updateDeck(svc))
		r.Delete("/", deleteDeck(svc))
	})
	r.Route("/areas/{id}", func(r chi.Router) {
		r.Get("/", getArea(svc))
		r.Patch("/", updateArea(svc))
		r.Delete("/", deleteArea(svc))
	})
	r.Route("/rooms/{id}", func(r chi.Router) {
		r.Get("/", getRoom(svc))
		r.Patch("/", updateRoom(svc))
		r.Delete("/", deleteRoom(svc))
	})
}

func listShips(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListShips(r.Context(), principal, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createShip(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		var req shipCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreateShip(r.Context(), principal, appship.ShipInput{
			Name: req.Name, ImoNumber: req.ImoNumber, FlagState: req.FlagState,
			Timezone: req.Timezone, Status: req.Status,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getShip(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetShip(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updateShip(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req shipUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdateShip(r.Context(), principal, id, appship.ShipUpdate{
			Name: req.Name, ImoNumber: req.ImoNumber, FlagState: req.FlagState,
			Timezone: req.Timezone, Status: req.Status,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteShip(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteShip(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listDecks(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListDecks(r.Context(), principal, shipID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createDeck(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req deckCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreateDeck(r.Context(), principal, shipID, appship.DeckInput{Level: *req.Level, Name: req.Name})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getDeck(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetDeck(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updateDeck(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req deckUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdateDeck(r.Context(), principal, id, appship.DeckUpdate{Level: req.Level, Name: req.Name})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteDeck(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteDeck(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listAreas(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListAreas(r.Context(), principal, shipID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createArea(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req areaCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		deckID, err := httpx.ParseUUID(req.DeckID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreateArea(r.Context(), principal, shipID, appship.AreaInput{
			DeckID: deckID, Name: req.Name, AreaType: req.AreaType, Capacity: req.Capacity, Layout: req.Layout,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getArea(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetArea(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updateArea(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req areaUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdateArea(r.Context(), principal, id, appship.AreaUpdate{
			Name: req.Name, Capacity: req.Capacity, Layout: req.Layout,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteArea(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteArea(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listRooms(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListRooms(r.Context(), principal, shipID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createRoom(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req roomCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		deckID, err := httpx.ParseUUID(req.DeckID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		isBookable := true
		if req.IsBookable != nil {
			isBookable = *req.IsBookable
		}
		row, err := svc.CreateRoom(r.Context(), principal, shipID, appship.RoomInput{
			DeckID: deckID, Number: req.Number, RoomType: req.RoomType,
			Capacity: *req.Capacity, IsBookable: isBookable, Layout: req.Layout,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getRoom(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetRoom(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updateRoom(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req roomUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		input := appship.RoomUpdate{
			Number: req.Number, RoomType: req.RoomType, Capacity: req.Capacity,
			IsBookable: req.IsBookable, Layout: req.Layout,
		}
		if req.DeckID != nil {
			deckID, err := httpx.ParseUUID(*req.DeckID)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.DeckID = &deckID
		}
		row, err := svc.UpdateRoom(r.Context(), principal, id, input)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteRoom(svc appship.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteRoom(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}
