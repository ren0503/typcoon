package booking

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/audit"
	"github.com/ren0503/tycoon/internal/application/auth"
	apppartner "github.com/ren0503/tycoon/internal/application/partner"
	appship "github.com/ren0503/tycoon/internal/application/ship"
	appvoyage "github.com/ren0503/tycoon/internal/application/voyage"
	apperrors "github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/platform/helpers"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

type service struct {
	repo       Repository
	shipSvc    appship.Service
	voyageSvc  appvoyage.Service
	partnerSvc apppartner.Service
	accountSvc auth.AccountService
	logger     *slog.Logger
}

// New builds the booking service.
func New(repo Repository, ships appship.Service, voyages appvoyage.Service, partners apppartner.Service, accounts auth.AccountService, logger *slog.Logger) Service {
	return &service{
		repo:       repo,
		shipSvc:    ships,
		voyageSvc:  voyages,
		partnerSvc: partners,
		accountSvc: accounts,
		logger:     logger,
	}
}

func (s *service) CreateBooking(ctx context.Context, p auth.Principal, input BookingInput) (booking Booking, err error) {
	defer func() { s.logOperation(ctx, "CreateBooking", err) }()

	if err := s.validateBeforeCreate(ctx, p, input); err != nil {
		return Booking{}, err
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return Booking{}, err
	}
	defer tx.Rollback(ctx)

	if input.IdempotencyKey != "" && input.AccountID != nil {
		existing, err := tx.GetBookingByIdempotency(ctx, GetBookingByIdempotencyParams{
			AccountID:      helpers.OptionalUUID(input.AccountID),
			IdempotencyKey: &input.IdempotencyKey,
		})
		if err == nil {
			if existing.IsSame(input) {
				return existing, nil
			}
			return Booking{}, apperrors.Conflict(apperrors.CodeIdempotencyConflict, "Idempotency key was reused with a different payload.")
		}
		if !isNotFound(err) {
			return Booking{}, err
		}
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Booking{}, apperrors.Internal(err)
	}
	var key *string
	if input.IdempotencyKey != "" {
		key = &input.IdempotencyKey
	}
	row, err := tx.CreateBooking(ctx, input.toCreateParams(id, key))
	if err != nil {
		return Booking{}, err
	}
	if err := s.appendAudit(ctx, tx, p, "booking.created", "booking", row.ID, row.ShipID, row.VoyageID, map[string]any{
		"embark_seq": row.EmbarkSeq, "disembark_seq": row.DisembarkSeq,
	}); err != nil {
		return Booking{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Booking{}, err
	}
	return row, nil
}

func (s *service) GetBooking(ctx context.Context, p auth.Principal, id uuid.UUID) (booking Booking, err error) {
	defer func() { s.logOperation(ctx, "GetBooking", err) }()

	row, err := s.repo.GetBooking(ctx, id)
	if err != nil {
		return Booking{}, err
	}

	return row, nil
}

func (s *service) ListBookings(ctx context.Context, p auth.Principal, shipID *uuid.UUID, rawCursor string, limit int32) (data pagination.Page[Booking], err error) {
	defer func() { s.logOperation(ctx, "GetBooking", err) }()

	if shipID != nil {
		if err := s.requireShip(ctx, p, *shipID); err != nil {
			return pagination.Page[Booking]{}, err
		}
	}
	scope := "tenant:" + p.TenantID.String()
	if shipID != nil {
		scope = shipID.String()
	}
	if p.ActorType == auth.ActorSystem && shipID == nil {
		scope = "all"
	}
	parts := []string{"bookings", scope}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return pagination.Page[Booking]{}, err
	}
	var rows []Booking
	switch {
	case shipID != nil:
		rows, err = s.repo.ListBookings(ctx, ListBookingsParams{
			ShipID: helpers.NullUUID(*shipID), After: helpers.NullUUID(cursor.ID), PageLimit: limit + 1,
		})
	case p.ActorType == auth.ActorSystem:
		rows, err = s.repo.ListBookings(ctx, ListBookingsParams{After: helpers.NullUUID(cursor.ID), PageLimit: limit + 1})
	default:
		rows, err = s.repo.ListBookingsForTenant(ctx, ListBookingsForTenantParams{
			TenantID: p.TenantID, After: helpers.NullUUID(cursor.ID), PageLimit: limit + 1,
		})
	}
	if err != nil {
		return pagination.Page[Booking]{}, err
	}
	page := pagination.Page[Booking]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateBooking(ctx context.Context, p auth.Principal, id uuid.UUID, input BookingUpdate) (booking Booking, err error) {
	defer func() { s.logOperation(ctx, "UpdateBooking", err) }()

	current, err := s.GetBooking(ctx, p, id)
	if err != nil {
		return Booking{}, err
	}
	if input.RoomID != nil {
		roomShip, err := s.shipSvc.RoomShipID(ctx, *input.RoomID)
		if err != nil || roomShip != current.ShipID {
			return Booking{}, apperrors.Validation(apperrors.CodeInvalidReference, "Room does not belong to the ship.")
		}
		current.RoomID = helpers.OptionalUUID(input.RoomID)
	}

	if current.DisembarkSeq <= current.EmbarkSeq {
		return Booking{}, validation("Disembark sequence must be greater than embark sequence.")
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return Booking{}, err
	}

	defer tx.Rollback(ctx)
	row, err := tx.UpdateBooking(ctx, input.toUpdateParam(id, current))
	if err != nil {
		return Booking{}, err
	}
	if err := s.appendAudit(ctx, tx, p, "booking.updated", "booking", row.ID, row.ShipID, row.VoyageID, map[string]any{
		"embark_seq": row.EmbarkSeq, "disembark_seq": row.DisembarkSeq,
	}); err != nil {
		return Booking{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Booking{}, err
	}
	return row, nil
}

func (s *service) DeleteBooking(ctx context.Context, p auth.Principal, id uuid.UUID) (err error) {
	defer func() { s.logOperation(ctx, "DeleteBooking", err) }()

	current, err := s.GetBooking(ctx, p, id)
	if err != nil {
		return err
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := tx.SoftDeleteBooking(ctx, id); err != nil {
		return err
	}
	if err := s.appendAudit(ctx, tx, p, "booking.deleted", "booking", id, current.ShipID, current.VoyageID, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *service) BookingExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.repo.BookingExists(ctx, id)
}

func (s *service) CreateServiceBooking(ctx context.Context, p auth.Principal, input ServiceBookingInput) (sb ServiceBooking, err error) {
	defer func() { s.logOperation(ctx, "CreateServiceBooking", err) }()

	bookingRow, err := s.GetBooking(ctx, p, input.BookingID)
	if err != nil {
		return ServiceBooking{}, err
	}

	err = s.validateBeforeCreateSB(ctx, p, input)
	if err != nil {
		return ServiceBooking{}, err
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return ServiceBooking{}, err
	}
	defer tx.Rollback(ctx)

	if input.IdempotencyKey != "" {
		existing, err := tx.GetServiceBookingByIdempotency(ctx, GetServiceBookingByIdempotencyParams{
			BookingID: input.BookingID, IdempotencyKey: &input.IdempotencyKey,
		})
		if err == nil {
			if existing.IsSame(input) {
				return existing, nil
			}
			return ServiceBooking{}, apperrors.Conflict(apperrors.CodeIdempotencyConflict, "Idempotency key was reused with a different payload.")
		}
		if !isNotFound(err) {
			return ServiceBooking{}, err
		}
	}
	id, err := uuid.NewV7()
	if err != nil {
		return ServiceBooking{}, apperrors.Internal(err)
	}
	var key *string
	if input.IdempotencyKey != "" {
		key = &input.IdempotencyKey
	}
	row, err := tx.CreateServiceBooking(ctx, input.toCreateParam(id, key))
	if err != nil {
		return ServiceBooking{}, err
	}
	if err := s.appendAudit(ctx, tx, p, "service_booking.created", "service_booking", row.ID, bookingRow.ShipID, bookingRow.VoyageID, map[string]any{
		"booking_id": row.BookingID, "quantity": row.Quantity,
	}); err != nil {
		return ServiceBooking{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ServiceBooking{}, err
	}
	return row, nil
}

func (s *service) GetServiceBooking(ctx context.Context, p auth.Principal, id uuid.UUID) (sb ServiceBooking, err error) {
	defer func() { s.logOperation(ctx, "GetServiceBooking", err) }()

	row, err := s.repo.GetServiceBooking(ctx, id)
	if err != nil {
		return ServiceBooking{}, err
	}
	if _, err := s.GetBooking(ctx, p, row.BookingID); err != nil {
		return ServiceBooking{}, err
	}
	return row, nil
}

func (s *service) ListServiceBookings(ctx context.Context, p auth.Principal, bookingID uuid.UUID, rawCursor string, limit int32) (data pagination.Page[ServiceBooking], err error) {
	defer func() { s.logOperation(ctx, "ListServiceBookings", err) }()

	if _, err := s.GetBooking(ctx, p, bookingID); err != nil {
		return pagination.Page[ServiceBooking]{}, err
	}
	parts := []string{"service_bookings", bookingID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return pagination.Page[ServiceBooking]{}, err
	}
	rows, err := s.repo.ListServiceBookings(ctx, ListServiceBookingsParams{
		BookingID: helpers.NullUUID(bookingID), After: helpers.NullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return pagination.Page[ServiceBooking]{}, err
	}
	page := pagination.Page[ServiceBooking]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateServiceBooking(ctx context.Context, p auth.Principal, id uuid.UUID, input ServiceBookingUpdate) (boooking ServiceBooking, err error) {
	defer func() { s.logOperation(ctx, "UpdateServiceBooking", err) }()

	current, err := s.GetServiceBooking(ctx, p, id)
	if err != nil {
		return ServiceBooking{}, err
	}

	if input.Quantity != nil {
		if *input.Quantity <= 0 {
			return ServiceBooking{}, validation("Service booking quantity must be positive.")
		}
		current.Quantity = *input.Quantity
	}
	if input.Status != nil {
		if !isServiceBookingStatus(*input.Status) {
			return ServiceBooking{}, validation("Service booking status is not recognised.")
		}
		current.Status = *input.Status
	}

	bookingRow, err := s.GetBooking(ctx, p, current.BookingID)
	if err != nil {
		return ServiceBooking{}, err
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return ServiceBooking{}, err
	}
	defer tx.Rollback(ctx)
	row, err := tx.UpdateServiceBooking(ctx, input.toUpdateParam(id, current))
	if err != nil {
		return ServiceBooking{}, err
	}
	if err := s.appendAudit(ctx, tx, p, "service_booking.updated", "service_booking", row.ID, bookingRow.ShipID, bookingRow.VoyageID, map[string]any{
		"booking_id": row.BookingID, "status": row.Status, "quantity": row.Quantity,
	}); err != nil {
		return ServiceBooking{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ServiceBooking{}, err
	}
	return row, nil
}

func (s *service) DeleteServiceBooking(ctx context.Context, p auth.Principal, id uuid.UUID) (err error) {
	defer func() { s.logOperation(ctx, "UpdateServiceBooking", err) }()

	current, err := s.GetServiceBooking(ctx, p, id)
	if err != nil {
		return err
	}
	bookingRow, err := s.GetBooking(ctx, p, current.BookingID)
	if err != nil {
		return err
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := tx.SoftDeleteServiceBooking(ctx, id); err != nil {
		return err
	}
	if err := s.appendAudit(ctx, tx, p, "service_booking.deleted", "service_booking", id, bookingRow.ShipID, bookingRow.VoyageID, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *service) appendAudit(
	ctx context.Context,
	tx Tx,
	p auth.Principal,
	action, entity string,
	entityID, shipID, voyageID uuid.UUID,
	details map[string]any,
) error {
	payload := json.RawMessage(`{}`)
	if details != nil {
		encoded, err := json.Marshal(details)
		if err != nil {
			return apperrors.Internal(err)
		}
		payload = encoded
	}
	actorID := p.ActorID
	entityName := entity
	entityRef := entityID.String()
	entry := audit.Entry{
		Source: "cloud", ActorType: string(p.ActorType),
		Action: action, Entity: &entityName, EntityID: &entityRef, Details: payload,
	}
	if actorID != "" {
		entry.ActorID = &actorID
	}
	if shipID != uuid.Nil {
		entry.ShipID = &shipID
	}
	if voyageID != uuid.Nil {
		entry.VoyageID = &voyageID
	}
	return tx.AppendAudit(ctx, entry)
}

func (s *service) requireShip(ctx context.Context, p auth.Principal, shipID uuid.UUID) error {
	tenantID, err := s.shipSvc.TenantID(ctx, shipID)
	if err != nil {
		return err
	}
	if !p.CanAccessTenant(tenantID) || !p.CanAccessShip(shipID) {
		return apperrors.Forbidden("Booking access is not permitted.")
	}
	return nil
}

func (s *service) validateBeforeCreate(ctx context.Context, p auth.Principal, input BookingInput) error {
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return err
	}
	v, err := s.voyageSvc.Lookup(ctx, input.VoyageID)
	if err != nil {
		return err
	}
	if v.ShipID != input.ShipID {
		return apperrors.Validation(apperrors.CodeInvalidReference, "Voyage does not belong to the ship.")
	}
	if input.AccountID != nil {
		ok, err := s.accountSvc.Exists(ctx, *input.AccountID)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.Validation(apperrors.CodeInvalidReference, "Account does not exist.")
		}
	}
	if input.RoomID != nil {
		roomShip, err := s.shipSvc.RoomShipID(ctx, *input.RoomID)
		if err != nil {
			return apperrors.Validation(apperrors.CodeInvalidReference, "Room does not exist.")
		}
		if roomShip != input.ShipID {
			return apperrors.Validation(apperrors.CodeInvalidReference, "Room does not belong to the ship.")
		}
	}
	if input.EmbarkSeq < 0 || input.DisembarkSeq <= input.EmbarkSeq {
		return apperrors.Validation(apperrors.CodeValidationFailed, "Disembark sequence must be greater than embark sequence.")
	}
	return nil
}

func (s *service) validateBeforeCreateSB(ctx context.Context, p auth.Principal, input ServiceBookingInput) error {
	ok, err := s.partnerSvc.ServiceExists(ctx, input.PartnerServiceID)
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.Validation(apperrors.CodeInvalidReference, "Partner service does not exist.")
	}
	if input.SlotID != nil {
		ok, err := s.partnerSvc.SlotExists(ctx, *input.SlotID)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.Validation(apperrors.CodeInvalidReference, "Partner slot does not exist.")
		}
	}
	if input.Quantity <= 0 {
		return validation("Service booking quantity must be positive.")
	}

	return nil
}

func (s *service) logOperation(ctx context.Context, operation string, err error) {
	service := "booking"
	attrs := []any{"service", service, "operation", operation}
	if err != nil {
		s.logger.ErrorContext(ctx, "operation failed", append(attrs, "error", err.Error())...)
	} else {
		s.logger.InfoContext(ctx, "operation succeeded", attrs...)
	}
}
