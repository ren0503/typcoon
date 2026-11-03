package guest

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/audit"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/booking"
	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
	"github.com/ren0503/tycoon/internal/pagination"
)

type service struct {
	repo     *repo
	bookings booking.Service
	audit    audit.Appender
	cipher   DocumentCipher
}

// New builds the guest service.
func New(pool *pgxpool.Pool, bookings booking.Service, appender audit.Appender, documentCipher DocumentCipher) Service {
	return &service{repo: newRepo(pool), bookings: bookings, audit: appender, cipher: documentCipher}
}

func (s *service) Create(ctx context.Context, p auth.Principal, bookingID uuid.UUID, input Input) (View, error) {
	bookingRow, err := s.bookings.GetBooking(ctx, p, bookingID)
	if err != nil {
		return View{}, err
	}
	if strings.TrimSpace(input.FullName) == "" {
		return View{}, validation("Guest full name is required.")
	}
	if err := validateCiphertext(s.cipher, input.DocumentEnc, "document_enc"); err != nil {
		return View{}, err
	}
	if err := validateCiphertext(s.cipher, input.EmergencyContactEnc, "emergency_contact_enc"); err != nil {
		return View{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return View{}, apperrors.Internal(err)
	}
	tx, err := s.repo.begin(ctx)
	if err != nil {
		return View{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	row, err := sqlc.New(tx).CreateGuest(ctx, sqlc.CreateGuestParams{
		ID: id, BookingID: bookingID, FullName: strings.TrimSpace(input.FullName),
		DateOfBirth: dateParam(input.DateOfBirth), Nationality: trimPtr(input.Nationality),
		DocumentType: trimPtr(input.DocumentType), DocumentEnc: input.DocumentEnc,
		DocumentKeyID: trimPtr(input.DocumentKeyID), EmergencyContactEnc: input.EmergencyContactEnc,
		SpecialNeeds: trimPtr(input.SpecialNeeds), IsLead: input.IsLead, MusterStation: trimPtr(input.MusterStation),
	})
	if err != nil {
		return View{}, database.MapPgError(err)
	}
	if err := s.appendAudit(ctx, tx, p, "guest.created", "guest", row.ID, bookingRow.ShipID, bookingRow.VoyageID, map[string]any{
		"booking_id": row.BookingID, "is_lead": row.IsLead,
	}); err != nil {
		return View{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return View{}, database.MapPgError(err)
	}
	return toView(row), nil
}

func (s *service) Get(ctx context.Context, p auth.Principal, id uuid.UUID) (View, error) {
	row, err := s.repo.getGuest(ctx, id)
	if err != nil {
		return View{}, err
	}
	if _, err := s.bookings.GetBooking(ctx, p, row.BookingID); err != nil {
		return View{}, err
	}
	return toView(row), nil
}

func (s *service) List(ctx context.Context, p auth.Principal, bookingID uuid.UUID, rawCursor string, limit int32) (Page[View], error) {
	if _, err := s.bookings.GetBooking(ctx, p, bookingID); err != nil {
		return Page[View]{}, err
	}
	parts := []string{"guests", bookingID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[View]{}, err
	}
	rows, err := s.repo.listGuests(ctx, sqlc.ListGuestsParams{
		BookingID: bookingID, After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[View]{}, err
	}
	page := Page[View]{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: rows[len(rows)-1].ID}.Encode()
	}
	for _, row := range rows {
		page.Items = append(page.Items, toView(row))
	}
	return page, nil
}

func (s *service) Update(ctx context.Context, p auth.Principal, id uuid.UUID, input Update) (View, error) {
	current, err := s.repo.getGuest(ctx, id)
	if err != nil {
		return View{}, err
	}
	bookingRow, err := s.bookings.GetBooking(ctx, p, current.BookingID)
	if err != nil {
		return View{}, err
	}
	if input.FullName != nil {
		if strings.TrimSpace(*input.FullName) == "" {
			return View{}, validation("Guest full name is required.")
		}
		current.FullName = strings.TrimSpace(*input.FullName)
	}
	if input.SpecialNeeds != nil {
		current.SpecialNeeds = trimPtr(input.SpecialNeeds)
	}
	if input.IsLead != nil {
		current.IsLead = *input.IsLead
	}
	if input.MusterStation != nil {
		current.MusterStation = trimPtr(input.MusterStation)
	}
	if input.DocumentType != nil {
		current.DocumentType = trimPtr(input.DocumentType)
	}
	tx, err := s.repo.begin(ctx)
	if err != nil {
		return View{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	row, err := sqlc.New(tx).UpdateGuest(ctx, sqlc.UpdateGuestParams{
		FullName: current.FullName, DateOfBirth: current.DateOfBirth, Nationality: current.Nationality,
		DocumentType: current.DocumentType, DocumentEnc: current.DocumentEnc, DocumentKeyID: current.DocumentKeyID,
		EmergencyContactEnc: current.EmergencyContactEnc, SpecialNeeds: current.SpecialNeeds,
		IsLead: current.IsLead, MusterStation: current.MusterStation, ID: id,
	})
	if err != nil {
		return View{}, database.MapPgError(err)
	}
	if err := s.appendAudit(ctx, tx, p, "guest.updated", "guest", row.ID, bookingRow.ShipID, bookingRow.VoyageID, map[string]any{
		"booking_id": row.BookingID, "is_lead": row.IsLead,
	}); err != nil {
		return View{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return View{}, database.MapPgError(err)
	}
	return toView(row), nil
}

func (s *service) Delete(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	current, err := s.repo.getGuest(ctx, id)
	if err != nil {
		return err
	}
	bookingRow, err := s.bookings.GetBooking(ctx, p, current.BookingID)
	if err != nil {
		return err
	}
	tx, err := s.repo.begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	n, err := sqlc.New(tx).SoftDeleteGuest(ctx, id)
	if err != nil {
		return database.MapPgError(err)
	}
	if n == 0 {
		return database.MapPgError(pgx.ErrNoRows)
	}
	if err := s.appendAudit(ctx, tx, p, "guest.deleted", "guest", id, bookingRow.ShipID, bookingRow.VoyageID, nil); err != nil {
		return err
	}
	return database.MapPgError(tx.Commit(ctx))
}

func (s *service) IssueBoardingPass(ctx context.Context, p auth.Principal, bookingID uuid.UUID, guestID *uuid.UUID, expiresAt *time.Time) (BoardingPassView, error) {
	bookingRow, err := s.bookings.GetBooking(ctx, p, bookingID)
	if err != nil {
		return BoardingPassView{}, err
	}
	if guestID != nil {
		guestRow, err := s.repo.getGuest(ctx, *guestID)
		if err != nil {
			return BoardingPassView{}, err
		}
		if guestRow.BookingID != bookingID {
			return BoardingPassView{}, validation("Guest does not belong to the booking.")
		}
	}
	id, err := uuid.NewV7()
	if err != nil {
		return BoardingPassView{}, apperrors.Internal(err)
	}
	jti, err := uuid.NewV7()
	if err != nil {
		return BoardingPassView{}, apperrors.Internal(err)
	}
	expiry := time.Now().Add(24 * time.Hour)
	if expiresAt != nil {
		expiry = *expiresAt
	}
	if !expiry.After(time.Now()) {
		return BoardingPassView{}, validation("Boarding pass expiry must be in the future.")
	}
	tx, err := s.repo.begin(ctx)
	if err != nil {
		return BoardingPassView{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	row, err := sqlc.New(tx).CreateBoardingPass(ctx, sqlc.CreateBoardingPassParams{
		ID: id, BookingID: bookingID, GuestID: optionalUUID(guestID), Jti: jti, ExpiresAt: expiry,
	})
	if err != nil {
		return BoardingPassView{}, database.MapPgError(err)
	}
	if err := s.appendAudit(ctx, tx, p, "boarding_pass.issued", "boarding_pass", row.ID, bookingRow.ShipID, bookingRow.VoyageID, map[string]any{
		"booking_id": row.BookingID,
	}); err != nil {
		return BoardingPassView{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BoardingPassView{}, database.MapPgError(err)
	}
	return toBoardingPassView(row), nil
}

func (s *service) GetBoardingPass(ctx context.Context, p auth.Principal, id uuid.UUID) (BoardingPassView, error) {
	row, err := s.repo.getBoardingPass(ctx, id)
	if err != nil {
		return BoardingPassView{}, err
	}
	if _, err := s.bookings.GetBooking(ctx, p, row.BookingID); err != nil {
		return BoardingPassView{}, err
	}
	return toBoardingPassView(row), nil
}

func (s *service) ListBoardingPasses(ctx context.Context, p auth.Principal, bookingID uuid.UUID, rawCursor string, limit int32) (Page[BoardingPassView], error) {
	if _, err := s.bookings.GetBooking(ctx, p, bookingID); err != nil {
		return Page[BoardingPassView]{}, err
	}
	parts := []string{"boarding_passes", bookingID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[BoardingPassView]{}, err
	}
	rows, err := s.repo.listBoardingPasses(ctx, sqlc.ListBoardingPassesParams{
		BookingID: bookingID, After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[BoardingPassView]{}, err
	}
	page := Page[BoardingPassView]{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: rows[len(rows)-1].ID}.Encode()
	}
	for _, row := range rows {
		page.Items = append(page.Items, toBoardingPassView(row))
	}
	return page, nil
}

func (s *service) RevokeBoardingPass(ctx context.Context, p auth.Principal, id uuid.UUID) (BoardingPassView, error) {
	current, err := s.repo.getBoardingPass(ctx, id)
	if err != nil {
		return BoardingPassView{}, err
	}
	bookingRow, err := s.bookings.GetBooking(ctx, p, current.BookingID)
	if err != nil {
		return BoardingPassView{}, err
	}
	tx, err := s.repo.begin(ctx)
	if err != nil {
		return BoardingPassView{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	row, err := sqlc.New(tx).RevokeBoardingPass(ctx, id)
	if err != nil {
		return BoardingPassView{}, database.MapPgError(err)
	}
	if err := s.appendAudit(ctx, tx, p, "boarding_pass.revoked", "boarding_pass", row.ID, bookingRow.ShipID, bookingRow.VoyageID, map[string]any{
		"booking_id": row.BookingID,
	}); err != nil {
		return BoardingPassView{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BoardingPassView{}, database.MapPgError(err)
	}
	return toBoardingPassView(row), nil
}

func (s *service) appendAudit(ctx context.Context, tx pgx.Tx, p auth.Principal, action, entity string, entityID, shipID, voyageID uuid.UUID, details map[string]any) error {
	if s.audit == nil {
		return nil
	}
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
		Source: "cloud", ActorType: string(p.ActorType), Action: action,
		Entity: &entityName, EntityID: &entityRef, Details: payload,
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
	return s.audit.AppendTx(ctx, tx, entry)
}

// BoardingPassView is the boarding pass response payload.
type BoardingPassView struct {
	ID        uuid.UUID  `json:"id"`
	BookingID uuid.UUID  `json:"booking_id"`
	GuestID   *uuid.UUID `json:"guest_id"`
	Jti       uuid.UUID  `json:"jti"`
	IssuedAt  time.Time  `json:"issued_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
}

func toView(row sqlc.Guest) View {
	return View{
		ID: row.ID, BookingID: row.BookingID, FullName: row.FullName, DateOfBirth: dateValue(row.DateOfBirth),
		Nationality: row.Nationality, DocumentType: row.DocumentType, DocumentKeyID: row.DocumentKeyID,
		HasDocument: len(row.DocumentEnc) > 0, HasEmergencyContact: len(row.EmergencyContactEnc) > 0,
		SpecialNeeds: row.SpecialNeeds, IsLead: row.IsLead, MusterStation: row.MusterStation,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version, DeletedAt: row.DeletedAt,
	}
}

func toBoardingPassView(row sqlc.BoardingPass) BoardingPassView {
	view := BoardingPassView{
		ID: row.ID, BookingID: row.BookingID, Jti: row.Jti, IssuedAt: row.IssuedAt,
		ExpiresAt: row.ExpiresAt, RevokedAt: row.RevokedAt,
	}
	if row.GuestID.Valid {
		guestID := row.GuestID.UUID
		view.GuestID = &guestID
	}
	return view
}

func dateParam(value *time.Time) pgtype.Date {
	if value == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *value, Valid: true}
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
