package booking

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/httpx"
)

type bookingCreateRequest struct {
	ShipID          string     `json:"ship_id" validate:"required,uuid"`
	VoyageID        string     `json:"voyage_id" validate:"required,uuid"`
	AccountID       *string    `json:"account_id" validate:"omitempty,uuid"`
	RoomID          *string    `json:"room_id" validate:"omitempty,uuid"`
	EmbarkSeq       int32      `json:"embark_seq" validate:"required,min=1"`
	DisembarkSeq    int32      `json:"disembark_seq" validate:"required,min=1"`
	ExpiresAt       *time.Time `json:"expires_at"`
	SpecialRequests *string    `json:"special_requests" validate:"omitempty,max=2000"`
}

type bookingUpdateRequest struct {
	RoomID          *string    `json:"room_id" validate:"omitempty,uuid"`
	EmbarkSeq       *int32     `json:"embark_seq" validate:"omitempty,min=1"`
	DisembarkSeq    *int32     `json:"disembark_seq" validate:"omitempty,min=1"`
	ExpiresAt       *time.Time `json:"expires_at"`
	SpecialRequests *string    `json:"special_requests" validate:"omitempty,max=2000"`
	Status          *string    `json:"status"`
}

type serviceBookingCreateRequest struct {
	BookingID        string  `json:"booking_id" validate:"required,uuid"`
	PartnerServiceID string  `json:"partner_service_id" validate:"required,uuid"`
	SlotID           *string `json:"slot_id" validate:"omitempty,uuid"`
	GuestID          *string `json:"guest_id" validate:"omitempty,uuid"`
	Quantity         *int16  `json:"quantity" validate:"omitempty,min=1"`
	PriceCents       *int64  `json:"price_cents"`
}

type serviceBookingUpdateRequest struct {
	Status   *string `json:"status" validate:"omitempty,oneof=pending confirmed rejected cancelled completed"`
	Quantity *int16  `json:"quantity" validate:"omitempty,min=1"`
}

// RegisterRoutes mounts booking routes on the router.
func RegisterRoutes(r chi.Router, svc Service) {
	r.Route("/bookings", func(r chi.Router) {
		r.Get("/", listBookings(svc))
		r.Post("/", createBooking(svc))
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getBooking(svc))
			r.Patch("/", updateBooking(svc))
			r.Delete("/", deleteBooking(svc))
		})
	})
	r.Route("/service-bookings", func(r chi.Router) {
		r.Get("/", listServiceBookings(svc))
		r.Post("/", createServiceBooking(svc))
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getServiceBooking(svc))
			r.Patch("/", updateServiceBooking(svc))
			r.Delete("/", deleteServiceBooking(svc))
		})
	})
}

func listBookings(svc Service) http.HandlerFunc {
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
		page, err := svc.ListBookings(r.Context(), principal, shipID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createBooking(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		var req bookingCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := httpx.ParseUUID(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		voyageID, err := httpx.ParseUUID(req.VoyageID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		accountID, err := optionalUUIDPtr(req.AccountID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		roomID, err := optionalUUIDPtr(req.RoomID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreateBooking(r.Context(), principal, BookingInput{
			ShipID: shipID, VoyageID: voyageID, AccountID: accountID, RoomID: roomID,
			EmbarkSeq: req.EmbarkSeq, DisembarkSeq: req.DisembarkSeq, ExpiresAt: req.ExpiresAt,
			SpecialRequests: req.SpecialRequests, IdempotencyKey: httpx.Header(r),
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getBooking(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetBooking(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updateBooking(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req bookingUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if req.Status != nil {
			httpx.WriteError(w, r, apperrors.Validation(apperrors.CodeBusinessRule, "Booking status is immutable in this phase."))
			return
		}
		roomID, err := optionalUUIDPtr(req.RoomID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdateBooking(r.Context(), principal, id, BookingUpdate{
			RoomID: roomID, EmbarkSeq: req.EmbarkSeq, DisembarkSeq: req.DisembarkSeq,
			ExpiresAt: req.ExpiresAt, SpecialRequests: req.SpecialRequests,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteBooking(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteBooking(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listServiceBookings(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		bookingID, err := httpx.QueryUUID(r, "booking_id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if bookingID == nil {
			httpx.WriteError(w, r, apperrors.Validation(apperrors.CodeValidationFailed, "Query parameter booking_id is required."))
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListServiceBookings(r.Context(), principal, *bookingID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createServiceBooking(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		var req serviceBookingCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		bookingID, err := httpx.ParseUUID(req.BookingID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		serviceID, err := httpx.ParseUUID(req.PartnerServiceID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		slotID, err := optionalUUIDPtr(req.SlotID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		guestID, err := optionalUUIDPtr(req.GuestID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		quantity := int16(1)
		if req.Quantity != nil {
			quantity = *req.Quantity
		}
		row, err := svc.CreateServiceBooking(r.Context(), principal, ServiceBookingInput{
			BookingID: bookingID, PartnerServiceID: serviceID, SlotID: slotID, GuestID: guestID,
			Quantity: quantity, PriceCents: req.PriceCents, IdempotencyKey: httpx.Header(r),
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getServiceBooking(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetServiceBooking(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updateServiceBooking(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req serviceBookingUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdateServiceBooking(r.Context(), principal, id, ServiceBookingUpdate{
			Status: req.Status, Quantity: req.Quantity,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteServiceBooking(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteServiceBooking(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func optionalUUIDPtr(raw *string) (*uuid.UUID, error) {
	if raw == nil {
		return nil, nil
	}
	parsed, err := httpx.ParseUUID(*raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
