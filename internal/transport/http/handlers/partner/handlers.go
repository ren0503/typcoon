package partner

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	apppartner "github.com/ren0503/tycoon/internal/application/partner"
	apperrors "github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/transport/http/httpx"
)

type partnerCreateRequest struct {
	Name         string  `json:"name" validate:"required,max=255"`
	BusinessType string  `json:"business_type" validate:"required,oneof=entertainment retail food_beverage spa excursion other"`
	ContactName  *string `json:"contact_name" validate:"omitempty,max=255"`
	ContactEmail *string `json:"contact_email" validate:"omitempty,email,max=255"`
	ContactPhone *string `json:"contact_phone" validate:"omitempty,max=32"`
}

type partnerUpdateRequest struct {
	Name         *string `json:"name" validate:"omitempty,min=1,max=255"`
	ContactName  *string `json:"contact_name" validate:"omitempty,max=255"`
	ContactEmail *string `json:"contact_email" validate:"omitempty,email,max=255"`
	ContactPhone *string `json:"contact_phone" validate:"omitempty,max=32"`
	Status       *string `json:"status" validate:"omitempty,oneof=pending active suspended terminated"`
}

type contractCreateRequest struct {
	ShipID          string  `json:"ship_id" validate:"required,uuid"`
	StartsOn        string  `json:"starts_on" validate:"required"`
	EndsOn          *string `json:"ends_on"`
	FeeModel        *string `json:"fee_model" validate:"omitempty,max=64"`
	FeeAmountCents  *int64  `json:"fee_amount_cents"`
	Currency        *string `json:"currency" validate:"omitempty,len=3"`
	TermsStorageKey *string `json:"terms_storage_key" validate:"omitempty,max=512"`
	Status          *string `json:"status" validate:"omitempty,oneof=draft active expired terminated"`
}

type contractUpdateRequest struct {
	ShipID          *string `json:"ship_id" validate:"omitempty,uuid"`
	StartsOn        *string `json:"starts_on"`
	EndsOn          *string `json:"ends_on"`
	FeeModel        *string `json:"fee_model" validate:"omitempty,max=64"`
	FeeAmountCents  *int64  `json:"fee_amount_cents"`
	Currency        *string `json:"currency" validate:"omitempty,len=3"`
	TermsStorageKey *string `json:"terms_storage_key" validate:"omitempty,max=512"`
	Status          *string `json:"status" validate:"omitempty,oneof=draft active expired terminated"`
}

type partnerStaffCreateRequest struct {
	ShipID    string  `json:"ship_id" validate:"required,uuid"`
	FullName  string  `json:"full_name" validate:"required,max=255"`
	RoleTitle *string `json:"role_title" validate:"omitempty,max=255"`
	Phone     *string `json:"phone" validate:"omitempty,max=32"`
	Status    *string `json:"status" validate:"omitempty,oneof=active inactive"`
	ValidFrom *string `json:"valid_from"`
	ValidTo   *string `json:"valid_to"`
}

type partnerStaffUpdateRequest struct {
	ShipID    *string `json:"ship_id" validate:"omitempty,uuid"`
	FullName  *string `json:"full_name" validate:"omitempty,min=1,max=255"`
	RoleTitle *string `json:"role_title" validate:"omitempty,max=255"`
	Phone     *string `json:"phone" validate:"omitempty,max=32"`
	Status    *string `json:"status" validate:"omitempty,oneof=active inactive"`
	ValidFrom *string `json:"valid_from"`
	ValidTo   *string `json:"valid_to"`
}

type partnerServiceCreateRequest struct {
	ShipID          string  `json:"ship_id" validate:"required,uuid"`
	AreaID          *string `json:"area_id" validate:"omitempty,uuid"`
	Name            string  `json:"name" validate:"required,max=255"`
	Category        string  `json:"category" validate:"required,max=64"`
	PriceCents      *int64  `json:"price_cents"`
	Currency        *string `json:"currency" validate:"omitempty,len=3"`
	RequiresSlot    *bool   `json:"requires_slot"`
	DefaultCapacity *int32  `json:"default_capacity"`
	IsActive        *bool   `json:"is_active"`
}

type partnerServiceUpdateRequest struct {
	ShipID          *string `json:"ship_id" validate:"omitempty,uuid"`
	AreaID          *string `json:"area_id" validate:"omitempty,uuid"`
	Name            *string `json:"name" validate:"omitempty,min=1,max=255"`
	Category        *string `json:"category" validate:"omitempty,min=1,max=64"`
	PriceCents      *int64  `json:"price_cents"`
	Currency        *string `json:"currency" validate:"omitempty,len=3"`
	RequiresSlot    *bool   `json:"requires_slot"`
	DefaultCapacity *int32  `json:"default_capacity"`
	IsActive        *bool   `json:"is_active"`
}

type slotCreateRequest struct {
	ShipID    string    `json:"ship_id" validate:"required,uuid"`
	AreaID    string    `json:"area_id" validate:"required,uuid"`
	VoyageID  *string   `json:"voyage_id" validate:"omitempty,uuid"`
	ServiceID *string   `json:"service_id" validate:"omitempty,uuid"`
	StartAt   time.Time `json:"start_at" validate:"required"`
	EndAt     time.Time `json:"end_at" validate:"required"`
	Capacity  *int32    `json:"capacity"`
}

type slotUpdateRequest struct {
	VoyageID     *string    `json:"voyage_id" validate:"omitempty,uuid"`
	ServiceID    *string    `json:"service_id" validate:"omitempty,uuid"`
	StartAt      *time.Time `json:"start_at"`
	EndAt        *time.Time `json:"end_at"`
	Capacity     *int32     `json:"capacity"`
	Status       *string    `json:"status" validate:"omitempty,oneof=requested approved rejected cancelled"`
	DecisionNote *string    `json:"decision_note" validate:"omitempty,max=1024"`
}

type contractResponse struct {
	apppartner.PartnerContract
	StartsOn *string `json:"starts_on"`
	EndsOn   *string `json:"ends_on"`
}

type partnerStaffResponse struct {
	apppartner.PartnerStaff
	ValidFrom *string `json:"valid_from"`
	ValidTo   *string `json:"valid_to"`
}

func contractDTO(row apppartner.PartnerContract) contractResponse {
	return contractResponse{PartnerContract: row, StartsOn: dateString(row.StartsOn), EndsOn: dateString(row.EndsOn)}
}

func partnerStaffDTO(row apppartner.PartnerStaff) partnerStaffResponse {
	return partnerStaffResponse{PartnerStaff: row, ValidFrom: dateString(row.ValidFrom), ValidTo: dateString(row.ValidTo)}
}

// RegisterRoutes mounts partner routes on the router.
func RegisterRoutes(r chi.Router, svc apppartner.Service) {
	r.Route("/partners", func(r chi.Router) {
		r.Get("/", listPartners(svc))
		r.Post("/", createPartner(svc))
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getPartner(svc))
			r.Patch("/", updatePartner(svc))
			r.Delete("/", deletePartner(svc))
		})
		r.Route("/{partnerId}/contracts", func(r chi.Router) {
			r.Get("/", listContracts(svc))
			r.Post("/", createContract(svc))
		})
		r.Route("/{partnerId}/staff", func(r chi.Router) {
			r.Get("/", listPartnerStaff(svc))
			r.Post("/", createPartnerStaff(svc))
		})
		r.Route("/{partnerId}/services", func(r chi.Router) {
			r.Get("/", listPartnerServices(svc))
			r.Post("/", createPartnerService(svc))
		})
		r.Route("/{partnerId}/slots", func(r chi.Router) {
			r.Get("/", listSlots(svc))
			r.Post("/", createSlot(svc))
		})
	})
	r.Route("/partner-contracts/{id}", func(r chi.Router) {
		r.Get("/", getContract(svc))
		r.Patch("/", updateContract(svc))
		r.Delete("/", deleteContract(svc))
	})
	r.Route("/partner-staff/{id}", func(r chi.Router) {
		r.Get("/", getPartnerStaff(svc))
		r.Patch("/", updatePartnerStaff(svc))
		r.Delete("/", deletePartnerStaff(svc))
	})
	r.Route("/partner-services/{id}", func(r chi.Router) {
		r.Get("/", getPartnerService(svc))
		r.Patch("/", updatePartnerService(svc))
		r.Delete("/", deletePartnerService(svc))
	})
	r.Route("/partner-slots/{id}", func(r chi.Router) {
		r.Get("/", getSlot(svc))
		r.Patch("/", updateSlot(svc))
		r.Delete("/", deleteSlot(svc))
	})
}

func listPartners(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListPartners(r.Context(), principal, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createPartner(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		var req partnerCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreatePartner(r.Context(), principal, apppartner.PartnerInput{
			Name: req.Name, BusinessType: req.BusinessType, ContactName: req.ContactName,
			ContactEmail: req.ContactEmail, ContactPhone: req.ContactPhone,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getPartner(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetPartner(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updatePartner(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req partnerUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdatePartner(r.Context(), principal, id, apppartner.PartnerUpdate{
			Name: req.Name, ContactName: req.ContactName, ContactEmail: req.ContactEmail,
			ContactPhone: req.ContactPhone, Status: req.Status,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deletePartner(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeletePartner(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listContracts(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		partnerID, err := httpx.PathID(r, "partnerId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListContracts(r.Context(), principal, partnerID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		items := make([]contractResponse, len(page.Items))
		for i, row := range page.Items {
			items[i] = contractDTO(row)
		}
		httpx.Page(w, items, page.NextCursor)
	}
}

func createContract(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		partnerID, err := httpx.PathID(r, "partnerId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req contractCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := httpx.ParseUUID(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		startsOn, err := parseDateValue(req.StartsOn)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		endsOn, err := parseOptionalDate(req.EndsOn)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		status := "draft"
		if req.Status != nil {
			status = *req.Status
		}
		row, err := svc.CreateContract(r.Context(), principal, partnerID, apppartner.ContractInput{
			ShipID: shipID, StartsOn: startsOn, EndsOn: endsOn, FeeModel: req.FeeModel,
			FeeAmountCents: req.FeeAmountCents, Currency: req.Currency, TermsStorageKey: req.TermsStorageKey, Status: status,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, contractDTO(row))
	}
}

func getContract(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetContract(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, contractDTO(row))
	}
}

func updateContract(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req contractUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		input := apppartner.ContractUpdate{
			FeeModel: req.FeeModel, FeeAmountCents: req.FeeAmountCents, Currency: req.Currency,
			TermsStorageKey: req.TermsStorageKey, Status: req.Status,
		}
		if req.ShipID != nil {
			shipID, err := httpx.ParseUUID(*req.ShipID)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.ShipID = &shipID
		}
		if req.StartsOn != nil {
			startsOn, err := parseDateValue(*req.StartsOn)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.StartsOn = startsOn
		}
		if req.EndsOn != nil {
			endsOn, err := parseOptionalDate(req.EndsOn)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			if endsOn != nil {
				input.EndsOn = endsOn
			}
		}
		row, err := svc.UpdateContract(r.Context(), principal, id, input)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, contractDTO(row))
	}
}

func deleteContract(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteContract(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listPartnerStaff(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		partnerID, err := httpx.PathID(r, "partnerId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListStaff(r.Context(), principal, partnerID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		items := make([]partnerStaffResponse, len(page.Items))
		for i, row := range page.Items {
			items[i] = partnerStaffDTO(row)
		}
		httpx.Page(w, items, page.NextCursor)
	}
}

func createPartnerStaff(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		partnerID, err := httpx.PathID(r, "partnerId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req partnerStaffCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := httpx.ParseUUID(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		from, err := parseOptionalDate(req.ValidFrom)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		to, err := parseOptionalDate(req.ValidTo)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		status := "active"
		if req.Status != nil {
			status = *req.Status
		}
		row, err := svc.CreateStaff(r.Context(), principal, partnerID, apppartner.StaffInput{
			ShipID: shipID, FullName: req.FullName, RoleTitle: req.RoleTitle, Phone: req.Phone,
			Status: status, ValidFrom: from, ValidTo: to,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, partnerStaffDTO(row))
	}
}

func getPartnerStaff(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetStaff(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, partnerStaffDTO(row))
	}
}

func updatePartnerStaff(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req partnerStaffUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		input := apppartner.StaffUpdate{
			FullName: req.FullName, RoleTitle: req.RoleTitle, Phone: req.Phone, Status: req.Status,
		}
		if req.ShipID != nil {
			shipID, err := httpx.ParseUUID(*req.ShipID)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.ShipID = &shipID
		}
		if req.ValidFrom != nil {
			from, err := parseOptionalDate(req.ValidFrom)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.ValidFrom = from
		}
		if req.ValidTo != nil {
			to, err := parseOptionalDate(req.ValidTo)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.ValidTo = to
		}
		row, err := svc.UpdateStaff(r.Context(), principal, id, input)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, partnerStaffDTO(row))
	}
}

func deletePartnerStaff(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteStaff(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listPartnerServices(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		partnerID, err := httpx.PathID(r, "partnerId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListServices(r.Context(), principal, partnerID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createPartnerService(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		partnerID, err := httpx.PathID(r, "partnerId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req partnerServiceCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := httpx.ParseUUID(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		areaID, err := optionalUUIDPtr(req.AreaID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		requiresSlot := false
		if req.RequiresSlot != nil {
			requiresSlot = *req.RequiresSlot
		}
		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}
		row, err := svc.CreateService(r.Context(), principal, partnerID, apppartner.ServiceInput{
			ShipID: shipID, AreaID: areaID, Name: req.Name, Category: req.Category, PriceCents: req.PriceCents,
			Currency: req.Currency, RequiresSlot: requiresSlot, DefaultCapacity: req.DefaultCapacity, IsActive: isActive,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getPartnerService(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetService(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updatePartnerService(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req partnerServiceUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		input := apppartner.ServiceUpdate{
			Name: req.Name, Category: req.Category, PriceCents: req.PriceCents, Currency: req.Currency,
			RequiresSlot: req.RequiresSlot, DefaultCapacity: req.DefaultCapacity, IsActive: req.IsActive,
		}
		if req.ShipID != nil {
			shipID, err := httpx.ParseUUID(*req.ShipID)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.ShipID = &shipID
		}
		if req.AreaID != nil {
			areaID, err := optionalUUIDPtr(req.AreaID)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.AreaID = areaID
		}
		row, err := svc.UpdateService(r.Context(), principal, id, input)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deletePartnerService(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteService(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listSlots(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		partnerID, err := httpx.PathID(r, "partnerId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListSlots(r.Context(), principal, partnerID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createSlot(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		partnerID, err := httpx.PathID(r, "partnerId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req slotCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := httpx.ParseUUID(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		areaID, err := httpx.ParseUUID(req.AreaID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		voyageID, err := optionalUUIDPtr(req.VoyageID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		serviceID, err := optionalUUIDPtr(req.ServiceID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreateSlot(r.Context(), principal, partnerID, apppartner.SlotInput{
			ShipID: shipID, AreaID: areaID, VoyageID: voyageID, ServiceID: serviceID,
			StartAt: req.StartAt, EndAt: req.EndAt, Capacity: req.Capacity,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getSlot(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetSlot(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updateSlot(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req slotUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		input := apppartner.SlotUpdate{
			StartAt: req.StartAt, EndAt: req.EndAt, Capacity: req.Capacity, Status: req.Status, DecisionNote: req.DecisionNote,
		}
		if req.VoyageID != nil {
			voyageID, err := optionalUUIDPtr(req.VoyageID)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.VoyageID = voyageID
		}
		if req.ServiceID != nil {
			serviceID, err := optionalUUIDPtr(req.ServiceID)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.ServiceID = serviceID
		}
		row, err := svc.UpdateSlot(r.Context(), principal, id, input)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteSlot(svc apppartner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteSlot(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func parseDateValue(raw string) (*time.Time, error) {
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, apperrors.Validation(apperrors.CodeValidationFailed, "Date must use the YYYY-MM-DD format.")
	}
	return &parsed, nil
}

func parseOptionalDate(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	return parseDateValue(*raw)
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

func dateString(d *time.Time) *string {
	if d == nil {
		return nil
	}
	value := d.Format("2006-01-02")
	return &value
}
