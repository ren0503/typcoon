package guest

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/httpx"
)

// RegisterRoutes mounts guest routes on the router.
func RegisterRoutes(r chi.Router, svc Service) {
	r.Route("/bookings/{bookingId}", func(r chi.Router) {
		r.Get("/guests", listGuests(svc))
		r.Post("/guests", createGuest(svc))
		r.Get("/boarding-passes", listBoardingPasses(svc))
		r.Post("/boarding-passes", issueBoardingPass(svc))
	})
	r.Route("/guests", func(r chi.Router) {
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getGuest(svc))
			r.Patch("/", updateGuest(svc))
			r.Delete("/", deleteGuest(svc))
		})
	})
	r.Route("/boarding-passes", func(r chi.Router) {
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getBoardingPass(svc))
			r.Post("/", revokeBoardingPass(svc))
		})
	})
}

func createGuest(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		bookingID, err := httpx.PathID(r, "bookingId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req guestCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if req.Document != nil && *req.Document != "" {
			httpx.WriteError(w, r, apperrors.Validation(apperrors.CodeBusinessRule, "Plaintext documents are not accepted; send opaque ciphertext in document_enc."))
			return
		}
		documentEnc, err := decodeCiphertext(req.DocumentEnc)
		if err != nil {
			httpx.WriteError(w, r, validation("document_enc must be base64-encoded ciphertext."))
			return
		}
		emergencyEnc, err := decodeCiphertext(req.EmergencyContactEnc)
		if err != nil {
			httpx.WriteError(w, r, validation("emergency_contact_enc must be base64-encoded ciphertext."))
			return
		}
		dateOfBirth, err := parseDate(req.DateOfBirth)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.Create(r.Context(), principal, bookingID, Input{
			FullName: req.FullName, DateOfBirth: dateOfBirth, Nationality: req.Nationality,
			DocumentType: req.DocumentType, DocumentEnc: documentEnc, DocumentKeyID: req.DocumentKeyID,
			EmergencyContactEnc: emergencyEnc, SpecialNeeds: req.SpecialNeeds, IsLead: req.IsLead,
			MusterStation: req.MusterStation,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func listGuests(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		bookingID, err := httpx.PathID(r, "bookingId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.List(r.Context(), principal, bookingID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func getGuest(svc Service) http.HandlerFunc {
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

func updateGuest(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req guestUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if req.Document != nil && *req.Document != "" {
			httpx.WriteError(w, r, apperrors.Validation(apperrors.CodeBusinessRule, "Plaintext documents are not accepted."))
			return
		}
		row, err := svc.Update(r.Context(), principal, id, Update{
			FullName: req.FullName, SpecialNeeds: req.SpecialNeeds, IsLead: req.IsLead,
			MusterStation: req.MusterStation, DocumentType: req.DocumentType,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteGuest(svc Service) http.HandlerFunc {
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

func issueBoardingPass(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		bookingID, err := httpx.PathID(r, "bookingId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req boardingPassIssueRequest
		if r.ContentLength > 0 {
			if err := httpx.Decode(w, r, &req); err != nil {
				httpx.WriteError(w, r, err)
				return
			}
		}
		var guestID *uuid.UUID
		if req.GuestID != nil {
			parsed, err := httpx.ParseUUID(*req.GuestID)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			guestID = &parsed
		}
		row, err := svc.IssueBoardingPass(r.Context(), principal, bookingID, guestID, req.ExpiresAt)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func listBoardingPasses(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		bookingID, err := httpx.PathID(r, "bookingId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListBoardingPasses(r.Context(), principal, bookingID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func getBoardingPass(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetBoardingPass(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func revokeBoardingPass(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.RevokeBoardingPass(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func parseDate(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", *raw)
	if err != nil {
		return nil, validation("date_of_birth must use YYYY-MM-DD format.")
	}
	return &parsed, nil
}

var _ = base64.StdEncoding
