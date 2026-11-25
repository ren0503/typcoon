package partner

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	appship "github.com/ren0503/tycoon/internal/application/ship"
	apperrors "github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/platform/helpers"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

type service struct {
	repo   Repository
	ships  appship.Service
	logger *slog.Logger
}

// New builds the partner service.
func New(repo Repository, ships appship.Service, logger *slog.Logger) Service {
	return &service{repo: repo, ships: ships, logger: logger}
}

func (s *service) CreatePartner(ctx context.Context, p auth.Principal, input PartnerInput) (partner Partner, err error) {
	defer func() { s.logOperation(ctx, "CreatePArtner", err) }()

	if err = s.requireTenant(p); err != nil {
		return Partner{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || !isBusinessType(input.BusinessType) {
		return Partner{}, helpers.Validation("Partner name and a recognised business type are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Partner{}, apperrors.Internal(err)
	}
	return s.repo.CreatePartner(ctx, CreatePartnerParams{
		ID: id, TenantID: p.TenantID, Name: name, BusinessType: input.BusinessType,
		ContactName: helpers.TrimPtr(input.ContactName), ContactEmail: helpers.TrimPtr(input.ContactEmail), ContactPhone: helpers.TrimPtr(input.ContactPhone),
	})
}

func (s *service) GetPartner(ctx context.Context, p auth.Principal, id uuid.UUID) (Partner, error) {
	return s.getPartnerScoped(ctx, p, id)
}

func (s *service) ListPartners(ctx context.Context, p auth.Principal, rawCursor string, limit int32) (Page[Partner], error) {
	parts := []string{"partners", p.TenantID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[Partner]{}, err
	}
	rows, err := s.repo.ListPartners(ctx, ListPartnersParams{
		TenantID: helpers.NullUUID(p.TenantID), After: helpers.NullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[Partner]{}, err
	}
	page := Page[Partner]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdatePartner(ctx context.Context, p auth.Principal, id uuid.UUID, input PartnerUpdate) (Partner, error) {
	current, err := s.GetPartner(ctx, p, id)
	if err != nil {
		return Partner{}, err
	}
	if input.Name != nil {
		if strings.TrimSpace(*input.Name) == "" {
			return Partner{}, helpers.Validation("Partner name must not be empty.")
		}
		current.Name = strings.TrimSpace(*input.Name)
	}
	if input.ContactName != nil {
		current.ContactName = helpers.TrimPtr(input.ContactName)
	}
	if input.ContactEmail != nil {
		current.ContactEmail = helpers.TrimPtr(input.ContactEmail)
	}
	if input.ContactPhone != nil {
		current.ContactPhone = helpers.TrimPtr(input.ContactPhone)
	}
	if input.Status != nil {
		if !isPartnerStatus(*input.Status) {
			return Partner{}, helpers.Validation("Partner status is not recognised.")
		}
		current.Status = *input.Status
	}
	return s.repo.UpdatePartner(ctx, UpdatePartnerParams{
		ID: id, Name: current.Name, BusinessType: current.BusinessType, ContactName: current.ContactName,
		ContactEmail: current.ContactEmail, ContactPhone: current.ContactPhone, Status: current.Status,
	})
}

func (s *service) DeletePartner(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetPartner(ctx, p, id); err != nil {
		return err
	}
	return s.repo.DeletePartner(ctx, id)
}

func (s *service) PartnerExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.repo.PartnerExists(ctx, PartnerExistsParams{ID: id})
}

func (s *service) ServiceExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.repo.PartnerServiceExists(ctx, id)
}

func (s *service) SlotExists(ctx context.Context, id uuid.UUID) (bool, error) {
	_, err := s.repo.GetSlot(ctx, id)
	if err != nil {
		if appErr, ok := apperrors.As(err); ok && appErr.HTTPStatus == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *service) CreateContract(ctx context.Context, p auth.Principal, partnerID uuid.UUID, input ContractInput) (PartnerContract, error) {
	if _, err := s.GetPartner(ctx, p, partnerID); err != nil {
		return PartnerContract{}, err
	}
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return PartnerContract{}, err
	}
	if input.StartsOn == nil {
		return PartnerContract{}, helpers.Validation("Contract start date is required.")
	}
	if input.EndsOn != nil && input.EndsOn.Before(*input.StartsOn) {
		return PartnerContract{}, helpers.Validation("Contract end date must not be before start date.")
	}
	if input.Status == "" {
		input.Status = "draft"
	}
	if !isContractStatus(input.Status) {
		return PartnerContract{}, helpers.Validation("Contract status is not recognised.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return PartnerContract{}, apperrors.Internal(err)
	}
	return s.repo.CreateContract(ctx, CreatePartnerContractParams{
		ID: id, ShipID: input.ShipID, PartnerID: partnerID, StartsOn: input.StartsOn, EndsOn: input.EndsOn,
		FeeModel: helpers.TrimPtr(input.FeeModel), FeeAmountCents: input.FeeAmountCents, Currency: helpers.TrimPtr(input.Currency),
		TermsStorageKey: helpers.TrimPtr(input.TermsStorageKey), Status: input.Status,
	})
}

func (s *service) GetContract(ctx context.Context, p auth.Principal, id uuid.UUID) (PartnerContract, error) {
	row, err := s.repo.GetContract(ctx, id)
	if err != nil {
		return PartnerContract{}, err
	}
	if _, err := s.GetPartner(ctx, p, row.PartnerID); err != nil {
		return PartnerContract{}, err
	}
	return row, nil
}

func (s *service) ListContracts(ctx context.Context, p auth.Principal, partnerID uuid.UUID, rawCursor string, limit int32) (Page[PartnerContract], error) {
	if _, err := s.GetPartner(ctx, p, partnerID); err != nil {
		return Page[PartnerContract]{}, err
	}
	parts := []string{"partner_contracts", partnerID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[PartnerContract]{}, err
	}
	rows, err := s.repo.ListContracts(ctx, ListPartnerContractsParams{
		PartnerID: partnerID, After: helpers.NullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[PartnerContract]{}, err
	}
	page := Page[PartnerContract]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateContract(ctx context.Context, p auth.Principal, id uuid.UUID, input ContractUpdate) (PartnerContract, error) {
	current, err := s.GetContract(ctx, p, id)
	if err != nil {
		return PartnerContract{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			return PartnerContract{}, err
		}
		current.ShipID = *input.ShipID
	}
	if input.StartsOn != nil {
		current.StartsOn = input.StartsOn
	}
	if input.EndsOn != nil {
		current.EndsOn = input.EndsOn
	}
	if input.FeeModel != nil {
		current.FeeModel = helpers.TrimPtr(input.FeeModel)
	}
	if input.FeeAmountCents != nil {
		current.FeeAmountCents = input.FeeAmountCents
	}
	if input.Currency != nil {
		current.Currency = helpers.TrimPtr(input.Currency)
	}
	if input.TermsStorageKey != nil {
		current.TermsStorageKey = helpers.TrimPtr(input.TermsStorageKey)
	}
	if input.Status != nil {
		if !isContractStatus(*input.Status) {
			return PartnerContract{}, helpers.Validation("Contract status is not recognised.")
		}
		current.Status = *input.Status
	}
	if current.StartsOn == nil || (current.EndsOn != nil && current.EndsOn.Before(*current.StartsOn)) {
		return PartnerContract{}, helpers.Validation("Contract dates are invalid.")
	}
	return s.repo.UpdateContract(ctx, UpdatePartnerContractParams{
		ID: id, ShipID: current.ShipID, StartsOn: current.StartsOn, EndsOn: current.EndsOn,
		FeeModel: current.FeeModel, FeeAmountCents: current.FeeAmountCents, Currency: current.Currency,
		TermsStorageKey: current.TermsStorageKey, Status: current.Status,
	})
}

func (s *service) DeleteContract(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetContract(ctx, p, id); err != nil {
		return err
	}
	return s.repo.DeleteContract(ctx, id)
}

func (s *service) CreateStaff(ctx context.Context, p auth.Principal, partnerID uuid.UUID, input StaffInput) (PartnerStaff, error) {
	if _, err := s.GetPartner(ctx, p, partnerID); err != nil {
		return PartnerStaff{}, err
	}
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return PartnerStaff{}, err
	}
	name := strings.TrimSpace(input.FullName)
	if name == "" {
		return PartnerStaff{}, helpers.Validation("Partner staff name is required.")
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if !isPartnerStaffStatus(input.Status) {
		return PartnerStaff{}, helpers.Validation("Partner staff status is not recognised.")
	}
	if err := helpers.ValidateDateRange(input.ValidFrom, input.ValidTo); err != nil {
		return PartnerStaff{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return PartnerStaff{}, apperrors.Internal(err)
	}
	return s.repo.CreateStaff(ctx, CreatePartnerStaffParams{
		ID: id, ShipID: input.ShipID, PartnerID: partnerID, FullName: name, RoleTitle: helpers.TrimPtr(input.RoleTitle),
		Phone: helpers.TrimPtr(input.Phone), Status: input.Status, ValidFrom: input.ValidFrom, ValidTo: input.ValidTo,
	})
}

func (s *service) GetStaff(ctx context.Context, p auth.Principal, id uuid.UUID) (PartnerStaff, error) {
	row, err := s.repo.GetStaff(ctx, id)
	if err != nil {
		return PartnerStaff{}, err
	}
	if _, err := s.GetPartner(ctx, p, row.PartnerID); err != nil {
		return PartnerStaff{}, err
	}
	return row, nil
}

func (s *service) ListStaff(ctx context.Context, p auth.Principal, partnerID uuid.UUID, rawCursor string, limit int32) (Page[PartnerStaff], error) {
	if _, err := s.GetPartner(ctx, p, partnerID); err != nil {
		return Page[PartnerStaff]{}, err
	}
	parts := []string{"partner_staff", partnerID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[PartnerStaff]{}, err
	}
	rows, err := s.repo.ListStaff(ctx, ListPartnerStaffParams{PartnerID: partnerID, After: helpers.NullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[PartnerStaff]{}, err
	}
	page := Page[PartnerStaff]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateStaff(ctx context.Context, p auth.Principal, id uuid.UUID, input StaffUpdate) (PartnerStaff, error) {
	current, err := s.GetStaff(ctx, p, id)
	if err != nil {
		return PartnerStaff{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			return PartnerStaff{}, err
		}
		current.ShipID = *input.ShipID
	}
	if input.FullName != nil {
		if strings.TrimSpace(*input.FullName) == "" {
			return PartnerStaff{}, helpers.Validation("Partner staff name must not be empty.")
		}
		current.FullName = strings.TrimSpace(*input.FullName)
	}
	if input.RoleTitle != nil {
		current.RoleTitle = helpers.TrimPtr(input.RoleTitle)
	}
	if input.Phone != nil {
		current.Phone = helpers.TrimPtr(input.Phone)
	}
	if input.Status != nil {
		if !isPartnerStaffStatus(*input.Status) {
			return PartnerStaff{}, helpers.Validation("Partner staff status is not recognised.")
		}
		current.Status = *input.Status
	}
	if input.ValidFrom != nil {
		current.ValidFrom = input.ValidFrom
	}
	if input.ValidTo != nil {
		current.ValidTo = input.ValidTo
	}
	if err := helpers.ValidateDateRange(current.ValidFrom, current.ValidTo); err != nil {
		return PartnerStaff{}, err
	}
	return s.repo.UpdateStaff(ctx, UpdatePartnerStaffParams{
		ID: id, ShipID: current.ShipID, FullName: current.FullName, RoleTitle: current.RoleTitle,
		Phone: current.Phone, Status: current.Status, ValidFrom: current.ValidFrom, ValidTo: current.ValidTo,
	})
}

func (s *service) DeleteStaff(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetStaff(ctx, p, id); err != nil {
		return err
	}
	return s.repo.DeleteStaff(ctx, id)
}

func (s *service) CreateService(ctx context.Context, p auth.Principal, partnerID uuid.UUID, input ServiceInput) (PartnerService, error) {
	if _, err := s.GetPartner(ctx, p, partnerID); err != nil {
		return PartnerService{}, err
	}
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return PartnerService{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || strings.TrimSpace(input.Category) == "" {
		return PartnerService{}, helpers.Validation("Service name and category are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return PartnerService{}, apperrors.Internal(err)
	}
	return s.repo.CreateService(ctx, CreatePartnerServiceParams{
		ID: id, ShipID: input.ShipID, PartnerID: partnerID, AreaID: helpers.OptionalUUID(input.AreaID), Name: name,
		Category: strings.TrimSpace(input.Category), PriceCents: input.PriceCents, Currency: helpers.TrimPtr(input.Currency),
		RequiresSlot: input.RequiresSlot, DefaultCapacity: input.DefaultCapacity, IsActive: input.IsActive,
	})
}

func (s *service) GetService(ctx context.Context, p auth.Principal, id uuid.UUID) (PartnerService, error) {
	row, err := s.repo.GetService(ctx, id)
	if err != nil {
		return PartnerService{}, err
	}
	if _, err := s.GetPartner(ctx, p, row.PartnerID); err != nil {
		return PartnerService{}, err
	}
	return row, nil
}

func (s *service) ListServices(ctx context.Context, p auth.Principal, partnerID uuid.UUID, rawCursor string, limit int32) (Page[PartnerService], error) {
	if _, err := s.GetPartner(ctx, p, partnerID); err != nil {
		return Page[PartnerService]{}, err
	}
	parts := []string{"partner_services", partnerID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[PartnerService]{}, err
	}
	rows, err := s.repo.ListServices(ctx, ListPartnerServicesParams{PartnerID: partnerID, After: helpers.NullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[PartnerService]{}, err
	}
	page := Page[PartnerService]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateService(ctx context.Context, p auth.Principal, id uuid.UUID, input ServiceUpdate) (PartnerService, error) {
	current, err := s.GetService(ctx, p, id)
	if err != nil {
		return PartnerService{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			return PartnerService{}, err
		}
		current.ShipID = *input.ShipID
	}
	if input.AreaID != nil {
		current.AreaID = helpers.OptionalUUID(input.AreaID)
	}
	if input.Name != nil {
		if strings.TrimSpace(*input.Name) == "" {
			return PartnerService{}, helpers.Validation("Service name must not be empty.")
		}
		current.Name = strings.TrimSpace(*input.Name)
	}
	if input.Category != nil {
		current.Category = strings.TrimSpace(*input.Category)
	}
	if input.PriceCents != nil {
		current.PriceCents = input.PriceCents
	}
	if input.Currency != nil {
		current.Currency = helpers.TrimPtr(input.Currency)
	}
	if input.RequiresSlot != nil {
		current.RequiresSlot = *input.RequiresSlot
	}
	if input.DefaultCapacity != nil {
		current.DefaultCapacity = input.DefaultCapacity
	}
	if input.IsActive != nil {
		current.IsActive = *input.IsActive
	}
	return s.repo.UpdateService(ctx, UpdatePartnerServiceParams{
		ID: id, ShipID: current.ShipID, AreaID: current.AreaID, Name: current.Name, Category: current.Category,
		PriceCents: current.PriceCents, Currency: current.Currency, RequiresSlot: current.RequiresSlot,
		DefaultCapacity: current.DefaultCapacity, IsActive: current.IsActive,
	})
}

func (s *service) DeleteService(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetService(ctx, p, id); err != nil {
		return err
	}
	return s.repo.DeleteService(ctx, id)
}

func (s *service) CreateSlot(ctx context.Context, p auth.Principal, partnerID uuid.UUID, input SlotInput) (PartnerSlot, error) {
	if _, err := s.GetPartner(ctx, p, partnerID); err != nil {
		return PartnerSlot{}, err
	}
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return PartnerSlot{}, err
	}
	if !input.EndAt.After(input.StartAt) {
		return PartnerSlot{}, helpers.Validation("Slot end must be after start.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return PartnerSlot{}, apperrors.Internal(err)
	}
	return s.repo.CreateSlot(ctx, CreatePartnerSlotParams{
		ID: id, ShipID: input.ShipID, PartnerID: partnerID, AreaID: input.AreaID,
		VoyageID: helpers.OptionalUUID(input.VoyageID), ServiceID: helpers.OptionalUUID(input.ServiceID),
		StartAt: input.StartAt, EndAt: input.EndAt, Capacity: input.Capacity,
	})
}

func (s *service) GetSlot(ctx context.Context, p auth.Principal, id uuid.UUID) (PartnerSlot, error) {
	row, err := s.repo.GetSlot(ctx, id)
	if err != nil {
		return PartnerSlot{}, err
	}
	if _, err := s.GetPartner(ctx, p, row.PartnerID); err != nil {
		return PartnerSlot{}, err
	}
	return row, nil
}

func (s *service) ListSlots(ctx context.Context, p auth.Principal, partnerID uuid.UUID, rawCursor string, limit int32) (Page[PartnerSlot], error) {
	if _, err := s.GetPartner(ctx, p, partnerID); err != nil {
		return Page[PartnerSlot]{}, err
	}
	parts := []string{"partner_slots", partnerID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[PartnerSlot]{}, err
	}
	rows, err := s.repo.ListSlots(ctx, ListPartnerSlotsParams{PartnerID: partnerID, After: helpers.NullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[PartnerSlot]{}, err
	}
	page := Page[PartnerSlot]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateSlot(ctx context.Context, p auth.Principal, id uuid.UUID, input SlotUpdate) (PartnerSlot, error) {
	current, err := s.GetSlot(ctx, p, id)
	if err != nil {
		return PartnerSlot{}, err
	}
	if input.VoyageID != nil {
		current.VoyageID = helpers.OptionalUUID(input.VoyageID)
	}
	if input.ServiceID != nil {
		current.ServiceID = helpers.OptionalUUID(input.ServiceID)
	}
	if input.StartAt != nil {
		current.StartAt = *input.StartAt
	}
	if input.EndAt != nil {
		current.EndAt = *input.EndAt
	}
	if input.Capacity != nil {
		current.Capacity = input.Capacity
	}
	if input.DecisionNote != nil {
		current.DecisionNote = helpers.TrimPtr(input.DecisionNote)
	}
	if input.Status != nil {
		if !isSlotStatus(*input.Status) {
			return PartnerSlot{}, helpers.Validation("Slot status is not recognised.")
		}
		if *input.Status == "approved" || *input.Status == "rejected" {
			now := time.Now().UTC()
			current.DecidedAt = &now
			if actor, err := uuid.Parse(p.ActorID); err == nil {
				current.DecidedBy = uuid.NullUUID{UUID: actor, Valid: true}
			}
		}
		current.Status = *input.Status
	}
	if !current.EndAt.After(current.StartAt) {
		return PartnerSlot{}, helpers.Validation("Slot end must be after start.")
	}
	return s.repo.UpdateSlot(ctx, UpdatePartnerSlotParams{
		ID: id, VoyageID: current.VoyageID, ServiceID: current.ServiceID, StartAt: current.StartAt,
		EndAt: current.EndAt, Capacity: current.Capacity, Status: current.Status,
		RequestedBy: current.RequestedBy, DecidedBy: current.DecidedBy, DecidedAt: current.DecidedAt,
		DecisionNote: current.DecisionNote,
	})
}

func (s *service) DeleteSlot(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetSlot(ctx, p, id); err != nil {
		return err
	}
	return s.repo.DeleteSlot(ctx, id)
}

func (s *service) getPartnerScoped(ctx context.Context, p auth.Principal, id uuid.UUID) (Partner, error) {
	row, err := s.repo.GetPartner(ctx, id)
	if err != nil {
		return Partner{}, err
	}
	if !p.CanAccessTenant(row.TenantID) {
		return Partner{}, apperrors.Forbidden("Partner access is not permitted.")
	}
	return row, nil
}

func (s *service) requireTenant(p auth.Principal) error {
	if p.TenantID == uuid.Nil {
		return helpers.Validation("A tenant is required.")
	}
	return nil
}

func (s *service) requireShip(ctx context.Context, p auth.Principal, shipID uuid.UUID) error {
	tenantID, err := s.ships.TenantID(ctx, shipID)
	if err != nil {
		return err
	}
	if !p.CanAccessTenant(tenantID) || !p.CanAccessShip(shipID) {
		return apperrors.Forbidden("Partner resource access is not permitted.")
	}
	return nil
}

func (s *service) logOperation(ctx context.Context, operation string, err error) {
	service := "partner"
	attrs := []any{"service", service, "operation", operation}
	if err != nil {
		s.logger.ErrorContext(ctx, "operation failed", append(attrs, "error", err.Error())...)
	} else {
		s.logger.InfoContext(ctx, "operation succeeded", attrs...)
	}
}
