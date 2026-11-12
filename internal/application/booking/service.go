package booking

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/audit"
	"github.com/ren0503/tycoon/internal/application/auth"
	apppartner "github.com/ren0503/tycoon/internal/application/partner"
	appship "github.com/ren0503/tycoon/internal/application/ship"
	appvoyage "github.com/ren0503/tycoon/internal/application/voyage"
	apperrors "github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

var serviceBookingStatuses = map[string]bool{
	"pending": true, "confirmed": true, "rejected": true, "cancelled": true, "completed": true,
}

type service struct {
	repo     Repository
	ships    appship.Service
	voyages  appvoyage.Service
	partners apppartner.Service
	accounts auth.AccountService
}

// New builds the booking service.
func New(repo Repository, ships appship.Service, voyages appvoyage.Service, partners apppartner.Service, accounts auth.AccountService) Service {
	return &service{repo: repo, ships: ships, voyages: voyages, partners: partners, accounts: accounts}
}

func (s *service) CreateBooking(ctx context.Context, p auth.Principal, input BookingInput) (Booking, error) {
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return Booking{}, err
	}
	v, err := s.voyages.Lookup(ctx, input.VoyageID)
	if err != nil {
		return Booking{}, err
	}
	if v.ShipID != input.ShipID {
		return Booking{}, apperrors.Validation(apperrors.CodeInvalidReference, "Voyage does not belong to the ship.")
	}
	if input.AccountID != nil {
		ok, err := s.accounts.Exists(ctx, *input.AccountID)
		if err != nil {
			return Booking{}, err
		}
		if !ok {
			return Booking{}, apperrors.Validation(apperrors.CodeInvalidReference, "Account does not exist.")
		}
	}
	if input.RoomID != nil {
		roomShip, err := s.ships.RoomShipID(ctx, *input.RoomID)
		if err != nil {
			return Booking{}, apperrors.Validation(apperrors.CodeInvalidReference, "Room does not exist.")
		}
		if roomShip != input.ShipID {
			return Booking{}, apperrors.Validation(apperrors.CodeInvalidReference, "Room does not belong to the ship.")
		}
	}
	if input.EmbarkSeq < 0 || input.DisembarkSeq <= input.EmbarkSeq {
		return Booking{}, validation("Disembark sequence must be greater than embark sequence.")
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return Booking{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if input.IdempotencyKey != "" && input.AccountID != nil {
		existing, err := tx.GetBookingByIdempotency(ctx, GetBookingByIdempotencyParams{
			AccountID: optionalUUID(input.AccountID), IdempotencyKey: &input.IdempotencyKey,
		})
		if err == nil {
			if sameBooking(existing, input) {
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
	row, err := tx.CreateBooking(ctx, CreateBookingParams{
		ID: id, ShipID: input.ShipID, VoyageID: input.VoyageID, AccountID: optionalUUID(input.AccountID),
		RoomID: optionalUUID(input.RoomID), EmbarkSeq: input.EmbarkSeq, DisembarkSeq: input.DisembarkSeq,
		ExpiresAt: input.ExpiresAt, IdempotencyKey: key, SpecialRequests: trimPtr(input.SpecialRequests),
	})
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

func (s *service) GetBooking(ctx context.Context, p auth.Principal, id uuid.UUID) (Booking, error) {
	row, err := s.repo.GetBooking(ctx, id)
	if err != nil {
		return Booking{}, err
	}
	if err := s.requireShip(ctx, p, row.ShipID); err != nil {
		return Booking{}, err
	}
	return row, nil
}

func (s *service) ListBookings(ctx context.Context, p auth.Principal, shipID *uuid.UUID, rawCursor string, limit int32) (Page[Booking], error) {
	if shipID != nil {
		if err := s.requireShip(ctx, p, *shipID); err != nil {
			return Page[Booking]{}, err
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
		return Page[Booking]{}, err
	}
	var rows []Booking
	switch {
	case shipID != nil:
		rows, err = s.repo.ListBookings(ctx, ListBookingsParams{
			ShipID: nullUUID(*shipID), After: nullUUID(cursor.ID), PageLimit: limit + 1,
		})
	case p.ActorType == auth.ActorSystem:
		rows, err = s.repo.ListBookings(ctx, ListBookingsParams{After: nullUUID(cursor.ID), PageLimit: limit + 1})
	default:
		rows, err = s.repo.ListBookingsForTenant(ctx, ListBookingsForTenantParams{
			TenantID: p.TenantID, After: nullUUID(cursor.ID), PageLimit: limit + 1,
		})
	}
	if err != nil {
		return Page[Booking]{}, err
	}
	page := Page[Booking]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateBooking(ctx context.Context, p auth.Principal, id uuid.UUID, input BookingUpdate) (Booking, error) {
	current, err := s.GetBooking(ctx, p, id)
	if err != nil {
		return Booking{}, err
	}
	if input.RoomID != nil {
		roomShip, err := s.ships.RoomShipID(ctx, *input.RoomID)
		if err != nil || roomShip != current.ShipID {
			return Booking{}, apperrors.Validation(apperrors.CodeInvalidReference, "Room does not belong to the ship.")
		}
		current.RoomID = optionalUUID(input.RoomID)
	}
	if input.EmbarkSeq != nil {
		current.EmbarkSeq = *input.EmbarkSeq
	}
	if input.DisembarkSeq != nil {
		current.DisembarkSeq = *input.DisembarkSeq
	}
	if input.ExpiresAt != nil {
		current.ExpiresAt = input.ExpiresAt
	}
	if input.SpecialRequests != nil {
		current.SpecialRequests = trimPtr(input.SpecialRequests)
	}
	if current.DisembarkSeq <= current.EmbarkSeq {
		return Booking{}, validation("Disembark sequence must be greater than embark sequence.")
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return Booking{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	row, err := tx.UpdateBooking(ctx, UpdateBookingParams{
		ID: id, RoomID: current.RoomID, EmbarkSeq: current.EmbarkSeq, DisembarkSeq: current.DisembarkSeq,
		ExpiresAt: current.ExpiresAt, SpecialRequests: current.SpecialRequests,
	})
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

func (s *service) DeleteBooking(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	current, err := s.GetBooking(ctx, p, id)
	if err != nil {
		return err
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
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

func (s *service) CreateServiceBooking(ctx context.Context, p auth.Principal, input ServiceBookingInput) (ServiceBooking, error) {
	bookingRow, err := s.GetBooking(ctx, p, input.BookingID)
	if err != nil {
		return ServiceBooking{}, err
	}
	ok, err := s.partners.ServiceExists(ctx, input.PartnerServiceID)
	if err != nil {
		return ServiceBooking{}, err
	}
	if !ok {
		return ServiceBooking{}, apperrors.Validation(apperrors.CodeInvalidReference, "Partner service does not exist.")
	}
	if input.SlotID != nil {
		ok, err := s.partners.SlotExists(ctx, *input.SlotID)
		if err != nil {
			return ServiceBooking{}, err
		}
		if !ok {
			return ServiceBooking{}, apperrors.Validation(apperrors.CodeInvalidReference, "Partner slot does not exist.")
		}
	}
	if input.Quantity <= 0 {
		return ServiceBooking{}, validation("Service booking quantity must be positive.")
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return ServiceBooking{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if input.IdempotencyKey != "" {
		existing, err := tx.GetServiceBookingByIdempotency(ctx, GetServiceBookingByIdempotencyParams{
			BookingID: input.BookingID, IdempotencyKey: &input.IdempotencyKey,
		})
		if err == nil {
			if sameServiceBooking(existing, input) {
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
	row, err := tx.CreateServiceBooking(ctx, CreateServiceBookingParams{
		ID: id, BookingID: input.BookingID, PartnerServiceID: input.PartnerServiceID,
		SlotID: optionalUUID(input.SlotID), GuestID: optionalUUID(input.GuestID), Quantity: input.Quantity,
		IdempotencyKey: key, PriceCents: input.PriceCents,
	})
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

func (s *service) GetServiceBooking(ctx context.Context, p auth.Principal, id uuid.UUID) (ServiceBooking, error) {
	row, err := s.repo.GetServiceBooking(ctx, id)
	if err != nil {
		return ServiceBooking{}, err
	}
	if _, err := s.GetBooking(ctx, p, row.BookingID); err != nil {
		return ServiceBooking{}, err
	}
	return row, nil
}

func (s *service) ListServiceBookings(ctx context.Context, p auth.Principal, bookingID uuid.UUID, rawCursor string, limit int32) (Page[ServiceBooking], error) {
	if _, err := s.GetBooking(ctx, p, bookingID); err != nil {
		return Page[ServiceBooking]{}, err
	}
	parts := []string{"service_bookings", bookingID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[ServiceBooking]{}, err
	}
	rows, err := s.repo.ListServiceBookings(ctx, ListServiceBookingsParams{
		BookingID: nullUUID(bookingID), After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[ServiceBooking]{}, err
	}
	page := Page[ServiceBooking]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateServiceBooking(ctx context.Context, p auth.Principal, id uuid.UUID, input ServiceBookingUpdate) (ServiceBooking, error) {
	current, err := s.GetServiceBooking(ctx, p, id)
	if err != nil {
		return ServiceBooking{}, err
	}
	if input.SlotID != nil {
		current.SlotID = optionalUUID(input.SlotID)
	}
	if input.GuestID != nil {
		current.GuestID = optionalUUID(input.GuestID)
	}
	if input.Quantity != nil {
		if *input.Quantity <= 0 {
			return ServiceBooking{}, validation("Service booking quantity must be positive.")
		}
		current.Quantity = *input.Quantity
	}
	if input.Status != nil {
		if !serviceBookingStatuses[*input.Status] {
			return ServiceBooking{}, validation("Service booking status is not recognised.")
		}
		current.Status = *input.Status
	}
	if input.PriceCents != nil {
		current.PriceCents = input.PriceCents
	}
	bookingRow, err := s.GetBooking(ctx, p, current.BookingID)
	if err != nil {
		return ServiceBooking{}, err
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return ServiceBooking{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	row, err := tx.UpdateServiceBooking(ctx, UpdateServiceBookingParams{
		ID: id, SlotID: current.SlotID, GuestID: current.GuestID, Quantity: current.Quantity,
		Status: current.Status, PriceCents: current.PriceCents,
	})
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

func (s *service) DeleteServiceBooking(ctx context.Context, p auth.Principal, id uuid.UUID) error {
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
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := tx.SoftDeleteServiceBooking(ctx, id); err != nil {
		return err
	}
	if err := s.appendAudit(ctx, tx, p, "service_booking.deleted", "service_booking", id, bookingRow.ShipID, bookingRow.VoyageID, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *service) appendAudit(ctx context.Context, tx Tx, p auth.Principal, action, entity string, entityID uuid.UUID, shipID, voyageID uuid.UUID, details map[string]any) error {
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
	tenantID, err := s.ships.TenantID(ctx, shipID)
	if err != nil {
		return err
	}
	if !p.CanAccessTenant(tenantID) || !p.CanAccessShip(shipID) {
		return apperrors.Forbidden("Booking access is not permitted.")
	}
	return nil
}

func sameBooking(existing Booking, input BookingInput) bool {
	if existing.ShipID != input.ShipID || existing.VoyageID != input.VoyageID {
		return false
	}
	if existing.RoomID != optionalUUID(input.RoomID) {
		return false
	}
	if existing.EmbarkSeq != input.EmbarkSeq || existing.DisembarkSeq != input.DisembarkSeq {
		return false
	}
	if !sameTime(existing.ExpiresAt, input.ExpiresAt) {
		return false
	}
	return sameString(existing.SpecialRequests, trimPtr(input.SpecialRequests))
}

func sameServiceBooking(existing ServiceBooking, input ServiceBookingInput) bool {
	if existing.PartnerServiceID != input.PartnerServiceID || existing.Quantity != input.Quantity {
		return false
	}
	if existing.SlotID != optionalUUID(input.SlotID) || existing.GuestID != optionalUUID(input.GuestID) {
		return false
	}
	if !sameInt64(existing.PriceCents, input.PriceCents) {
		return false
	}
	return true
}

func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func sameString(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func sameInt64(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func isNotFound(err error) bool {
	appErr, ok := apperrors.As(err)
	return ok && appErr.HTTPStatus == 404
}

func trimPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func validation(message string) error {
	return apperrors.Validation(apperrors.CodeValidationFailed, message)
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}

func optionalUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil || *id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}
