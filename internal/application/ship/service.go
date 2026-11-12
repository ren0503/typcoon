package ship

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

var (
	roomTypes = map[string]bool{
		"inside": true, "oceanview": true, "balcony": true,
		"suite": true, "crew": true, "other": true,
	}
	areaTypes = map[string]bool{
		"restaurant": true, "lobby": true, "open_deck": true, "theatre": true,
		"shop": true, "corridor": true, "stairwell": true, "spa": true,
		"pool": true, "other": true,
	}
	shipStatuses = map[string]bool{"active": true, "maintenance": true, "retired": true}
)

type service struct {
	repo   Repository
	logger *slog.Logger
}

// New builds the ship service from an injected repository.
func New(repo Repository, logger *slog.Logger) Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &service{repo: repo, logger: logger.With("component", "ship_service")}
}

func (s *service) CreateShip(ctx context.Context, p auth.Principal, input ShipInput) (Ship, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		s.logger.ErrorContext(ctx, "validation failed: ship name is required", "principal", p, "input", input)
		return Ship{}, validation("Ship name is required.")
	}
	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	if err := validateTimezone(timezone); err != nil {
		s.logger.ErrorContext(ctx, "validation failed: invalid timezone", "principal", p, "input", input, "error", err)
		return Ship{}, err
	}
	tenantID := p.TenantID
	if tenantID == uuid.Nil {
		s.logger.ErrorContext(ctx, "validation failed: tenant is required", "principal", p, "input", input)
		return Ship{}, apperrors.Validation(apperrors.CodeValidationFailed, "A tenant is required to create a ship.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate UUID for new ship", "principal", p, "input", input, "error", err)
		return Ship{}, apperrors.Internal(err)
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "active"
	}
	if !shipStatuses[status] {
		s.logger.ErrorContext(ctx, "validation failed: unrecognized ship status", "principal", p, "input", input)
		return Ship{}, validation("Ship status is not recognised.")
	}
	ship, err := s.repo.CreateShip(ctx, CreateShipParams{
		ID: id, TenantID: tenantID, Name: name,
		ImoNumber: trimPtr(input.ImoNumber), FlagState: trimPtr(input.FlagState), Timezone: timezone,
		Status: status,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create ship", "principal", p, "input", input, "error", err)
		return Ship{}, err
	}
	return ship, nil
}

func (s *service) GetShip(ctx context.Context, p auth.Principal, id uuid.UUID) (Ship, error) {
	return s.getShipScoped(ctx, p, id)
}

func (s *service) ListShips(ctx context.Context, p auth.Principal, rawCursor string, limit int32) (Page[Ship], error) {
	tenantID := p.TenantID
	if p.ActorType == auth.ActorSystem {
		tenantID = uuid.Nil
	}
	parts := []string{"ships", tenantID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[Ship]{}, err
	}
	rows, err := s.repo.ListShips(ctx, ListShipsParams{
		TenantID: nullUUID(tenantID), After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[Ship]{}, err
	}
	page := Page[Ship]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateShip(ctx context.Context, p auth.Principal, id uuid.UUID, input ShipUpdate) (Ship, error) {
	current, err := s.getShipScoped(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "ship not found or access denied", "principal", p, "ship_id", id, "error", err)
		return Ship{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			s.logger.ErrorContext(ctx, "validation failed: ship name is required", "principal", p, "input", input)
			return Ship{}, validation("Ship name must not be empty.")
		}
		current.Name = name
	}
	if input.Timezone != nil {
		if err := validateTimezone(*input.Timezone); err != nil {
			s.logger.ErrorContext(ctx, "validation failed: invalid timezone", "principal", p, "input", input, "error", err)
			return Ship{}, err
		}
		current.Timezone = strings.TrimSpace(*input.Timezone)
	}
	if input.ImoNumber != nil {
		current.ImoNumber = trimPtr(input.ImoNumber)
	}
	if input.FlagState != nil {
		current.FlagState = trimPtr(input.FlagState)
	}
	if input.Status != nil {
		if !shipStatuses[*input.Status] {
			s.logger.ErrorContext(ctx, "validation failed: unrecognized ship status", "principal", p, "input", input)
			return Ship{}, validation("Ship status is not recognised.")
		}
		current.Status = *input.Status
	}
	ship, err := s.repo.UpdateShip(ctx, UpdateShipParams{
		ID: current.ID, Name: current.Name, ImoNumber: current.ImoNumber,
		FlagState: current.FlagState, Timezone: current.Timezone, Status: current.Status,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update ship", "principal", p, "input", input, "error", err)
		return Ship{}, err
	}
	return ship, nil
}

func (s *service) DeleteShip(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.getShipScoped(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "ship not found or access denied", "principal", p, "ship_id", id, "error", err)
		return err
	}
	return s.repo.DeleteShip(ctx, id)
}

func (s *service) CreateDeck(ctx context.Context, p auth.Principal, shipID uuid.UUID, input DeckInput) (Deck, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		s.logger.ErrorContext(ctx, "ship not found or access denied", "principal", p, "ship_id", shipID, "error", err)
		return Deck{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || input.Level < 0 {
		s.logger.ErrorContext(ctx, "validation failed: deck name is required and level must not be negative", "principal", p, "input", input)
		return Deck{}, validation("Deck name is required and level must not be negative.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate UUID for new deck", "principal", p, "input", input, "error", err)
		return Deck{}, apperrors.Internal(err)
	}
	deck, err := s.repo.CreateDeck(ctx, CreateDeckParams{ID: id, ShipID: shipID, Level: input.Level, Name: name})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create deck", "principal", p, "input", input, "error", err)
		return Deck{}, err
	}
	return deck, nil
}

func (s *service) GetDeck(ctx context.Context, p auth.Principal, id uuid.UUID) (Deck, error) {
	deck, err := s.repo.GetDeck(ctx, id)
	if err != nil {
		return Deck{}, err
	}
	if _, err := s.getShipScoped(ctx, p, deck.ShipID); err != nil {
		return Deck{}, err
	}
	return deck, nil
}

func (s *service) ListDecks(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[Deck], error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return Page[Deck]{}, err
	}
	parts := []string{"decks", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[Deck]{}, err
	}
	rows, err := s.repo.ListDecks(ctx, ListDecksParams{ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[Deck]{}, err
	}
	page := Page[Deck]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateDeck(ctx context.Context, p auth.Principal, id uuid.UUID, input DeckUpdate) (Deck, error) {
	current, err := s.GetDeck(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "deck not found or access denied", "principal", p, "deck_id", id, "error", err)
		return Deck{}, err
	}
	if input.Level != nil {
		if *input.Level < 0 {
			s.logger.ErrorContext(ctx, "validation failed: deck level must not be negative", "principal", p, "input", input)
			return Deck{}, validation("Deck level must not be negative.")
		}
		current.Level = *input.Level
	}
	if input.Name != nil {
		if strings.TrimSpace(*input.Name) == "" {
			s.logger.ErrorContext(ctx, "validation failed: deck name must not be empty", "principal", p, "input", input)
			return Deck{}, validation("Deck name must not be empty.")
		}
		current.Name = strings.TrimSpace(*input.Name)
	}
	deck, err := s.repo.UpdateDeck(ctx, UpdateDeckParams{ID: id, Level: current.Level, Name: current.Name})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update deck", "principal", p, "input", input, "error", err)
		return Deck{}, err
	}
	return deck, nil
}

func (s *service) DeleteDeck(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetDeck(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "deck not found or access denied", "principal", p, "deck_id", id, "error", err)
		return err
	}
	return s.repo.DeleteDeck(ctx, id)
}

func (s *service) CreateArea(ctx context.Context, p auth.Principal, shipID uuid.UUID, input AreaInput) (Area, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		s.logger.ErrorContext(ctx, "ship not found or access denied", "principal", p, "ship_id", shipID, "error", err)
		return Area{}, err
	}
	if err := s.requireDeck(ctx, shipID, input.DeckID); err != nil {
		s.logger.ErrorContext(ctx, "deck not found or access denied", "principal", p, "deck_id", input.DeckID, "error", err)
		return Area{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || !areaTypes[input.AreaType] {
		s.logger.ErrorContext(ctx, "validation failed: area name and type are required", "principal", p, "input", input)
		return Area{}, validation("Area name and a recognised area type are required.")
	}
	if input.Capacity != nil && *input.Capacity < 0 {
		s.logger.ErrorContext(ctx, "validation failed: area capacity must not be negative", "principal", p, "input", input)
		return Area{}, validation("Area capacity must not be negative.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate UUID for new area", "principal", p, "input", input, "error", err)
		return Area{}, apperrors.Internal(err)
	}
	area, err := s.repo.CreateArea(ctx, CreateAreaParams{
		ID: id, ShipID: shipID, DeckID: input.DeckID, Name: name,
		AreaType: input.AreaType, Capacity: input.Capacity, Layout: input.Layout,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create area", "principal", p, "input", input, "error", err)
		return Area{}, err
	}
	return area, nil
}

func (s *service) GetArea(ctx context.Context, p auth.Principal, id uuid.UUID) (Area, error) {
	area, err := s.repo.GetArea(ctx, id)
	if err != nil {
		return Area{}, err
	}
	if _, err := s.getShipScoped(ctx, p, area.ShipID); err != nil {
		return Area{}, err
	}
	return area, nil
}

func (s *service) ListAreas(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[Area], error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return Page[Area]{}, err
	}
	parts := []string{"areas", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[Area]{}, err
	}
	rows, err := s.repo.ListAreas(ctx, ListAreasParams{ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[Area]{}, err
	}
	page := Page[Area]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateArea(ctx context.Context, p auth.Principal, id uuid.UUID, input AreaUpdate) (Area, error) {
	current, err := s.GetArea(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "area not found or access denied", "principal", p, "area_id", id, "error", err)
		return Area{}, err
	}
	if input.Name != nil {
		if strings.TrimSpace(*input.Name) == "" {
			s.logger.ErrorContext(ctx, "validation failed: area name must not be empty", "principal", p, "input", input)
			return Area{}, validation("Area name must not be empty.")
		}
		current.Name = strings.TrimSpace(*input.Name)
	}
	if input.Capacity != nil {
		if *input.Capacity < 0 {
			s.logger.ErrorContext(ctx, "validation failed: area capacity must not be negative", "principal", p, "input", input)
			return Area{}, validation("Area capacity must not be negative.")
		}
		current.Capacity = input.Capacity
	}
	if input.Layout != nil {
		current.Layout = input.Layout
	}
	area, err := s.repo.UpdateArea(ctx, UpdateAreaParams{
		ID: id, DeckID: current.DeckID, Name: current.Name, AreaType: current.AreaType,
		Capacity: current.Capacity, Layout: current.Layout,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update area", "principal", p, "input", input, "error", err)
		return Area{}, err
	}
	return area, nil
}

func (s *service) DeleteArea(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetArea(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "area not found or access denied", "principal", p, "area_id", id, "error", err)
		return err
	}
	return s.repo.DeleteArea(ctx, id)
}

func (s *service) CreateRoom(ctx context.Context, p auth.Principal, shipID uuid.UUID, input RoomInput) (Room, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		s.logger.ErrorContext(ctx, "ship not found or access denied", "principal", p, "ship_id", shipID, "error", err)
		return Room{}, err
	}
	if err := s.requireDeck(ctx, shipID, input.DeckID); err != nil {
		s.logger.ErrorContext(ctx, "deck not found or access denied", "principal", p, "deck_id", input.DeckID, "error", err)
		return Room{}, err
	}
	number := strings.TrimSpace(input.Number)
	if number == "" || !roomTypes[input.RoomType] || input.Capacity <= 0 {
		s.logger.ErrorContext(ctx, "validation failed: room number, type, and positive capacity are required", "principal", p, "input", input)
		return Room{}, validation("Room number, a recognised room type, and positive capacity are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate UUID for new room", "principal", p, "input", input, "error", err)
		return Room{}, apperrors.Internal(err)
	}
	room, err := s.repo.CreateRoom(ctx, CreateRoomParams{
		ID: id, ShipID: shipID, DeckID: input.DeckID, Number: number, RoomType: input.RoomType,
		Capacity: input.Capacity, IsBookable: input.IsBookable, Layout: input.Layout,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create room", "principal", p, "input", input, "error", err)
		return Room{}, err
	}
	return room, nil
}

func (s *service) GetRoom(ctx context.Context, p auth.Principal, id uuid.UUID) (Room, error) {
	room, err := s.repo.GetRoom(ctx, id)
	if err != nil {
		return Room{}, err
	}
	if _, err := s.getShipScoped(ctx, p, room.ShipID); err != nil {
		return Room{}, err
	}
	return room, nil
}

func (s *service) ListRooms(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[Room], error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return Page[Room]{}, err
	}
	parts := []string{"rooms", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[Room]{}, err
	}
	rows, err := s.repo.ListRooms(ctx, ListRoomsParams{ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[Room]{}, err
	}
	page := Page[Room]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateRoom(ctx context.Context, p auth.Principal, id uuid.UUID, input RoomUpdate) (Room, error) {
	current, err := s.GetRoom(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "room not found or access denied", "principal", p, "room_id", id, "error", err)
		return Room{}, err
	}
	if input.DeckID != nil {
		if err := s.requireDeck(ctx, current.ShipID, *input.DeckID); err != nil {
			s.logger.ErrorContext(ctx, "validation failed: deck not found or access denied", "principal", p, "deck_id", *input.DeckID, "error", err)
			return Room{}, err
		}
		current.DeckID = *input.DeckID
	}
	if input.Number != nil {
		if strings.TrimSpace(*input.Number) == "" {
			s.logger.ErrorContext(ctx, "validation failed: room number must not be empty", "principal", p, "input", input)
			return Room{}, validation("Room number must not be empty.")
		}
		current.Number = strings.TrimSpace(*input.Number)
	}
	if input.RoomType != nil {
		if !roomTypes[*input.RoomType] {
			s.logger.ErrorContext(ctx, "validation failed: unrecognized room type", "principal", p, "input", input)
			return Room{}, validation("Room type is not recognised.")
		}
		current.RoomType = *input.RoomType
	}
	if input.Capacity != nil {
		if *input.Capacity <= 0 {
			s.logger.ErrorContext(ctx, "validation failed: room capacity must be positive", "principal", p, "input", input)
			return Room{}, validation("Room capacity must be positive.")
		}
		current.Capacity = *input.Capacity
	}
	if input.IsBookable != nil {
		current.IsBookable = *input.IsBookable
	}
	if input.Layout != nil {
		current.Layout = input.Layout
	}
	room, err := s.repo.UpdateRoom(ctx, UpdateRoomParams{
		ID: id, DeckID: current.DeckID, Number: current.Number, RoomType: current.RoomType,
		Capacity: current.Capacity, IsBookable: current.IsBookable, Layout: current.Layout,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update room", "principal", p, "input", input, "error", err)
		return Room{}, err
	}
	return room, nil
}

func (s *service) DeleteRoom(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetRoom(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "room not found or access denied", "principal", p, "room_id", id, "error", err)
		return err
	}
	return s.repo.DeleteRoom(ctx, id)
}

func (s *service) Exists(ctx context.Context, shipID uuid.UUID) (bool, error) {
	return s.repo.ShipExists(ctx, shipID, uuid.Nil)
}

func (s *service) TenantID(ctx context.Context, shipID uuid.UUID) (uuid.UUID, error) {
	row, err := s.repo.GetShip(ctx, shipID)
	if err != nil {
		return uuid.Nil, err
	}
	return row.TenantID, nil
}

func (s *service) RoomShipID(ctx context.Context, roomID uuid.UUID) (uuid.UUID, error) {
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return uuid.Nil, err
	}
	return row.ShipID, nil
}

func (s *service) getShipScoped(ctx context.Context, p auth.Principal, id uuid.UUID) (Ship, error) {
	row, err := s.repo.GetShip(ctx, id)
	if err != nil {
		return Ship{}, err
	}
	if !p.CanAccessTenant(row.TenantID) || !p.CanAccessShip(row.ID) {
		return Ship{}, apperrors.Forbidden("Ship access is not permitted.")
	}
	return row, nil
}

func (s *service) requireDeck(ctx context.Context, shipID, deckID uuid.UUID) error {
	ok, err := s.repo.DeckExistsInShip(ctx, deckID, shipID)
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.Validation(apperrors.CodeInvalidReference, "Deck does not exist on this ship.")
	}
	return nil
}

func validateTimezone(timezone string) error {
	if strings.TrimSpace(timezone) == "" {
		return validation("Time zone is required.")
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return validation("Time zone must be a valid IANA time zone.")
	}
	return nil
}

func validation(message string) error {
	return apperrors.Validation(apperrors.CodeValidationFailed, message)
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

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}
