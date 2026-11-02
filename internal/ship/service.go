package ship

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
	"github.com/ren0503/tycoon/internal/pagination"
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
	repo *repo
}

// New builds the ship service.
func New(pool *pgxpool.Pool) Service {
	return &service{repo: newRepo(pool)}
}

func (s *service) CreateShip(ctx context.Context, p auth.Principal, input ShipInput) (sqlc.Ship, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return sqlc.Ship{}, validation("Ship name is required.")
	}
	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	if err := validateTimezone(timezone); err != nil {
		return sqlc.Ship{}, err
	}
	tenantID := p.TenantID
	if tenantID == uuid.Nil {
		return sqlc.Ship{}, apperrors.Validation(apperrors.CodeValidationFailed, "A tenant is required to create a ship.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.Ship{}, apperrors.Internal(err)
	}
	return s.repo.createShip(ctx, sqlc.CreateShipParams{
		ID: id, TenantID: tenantID, Name: name,
		ImoNumber: trimPtr(input.ImoNumber), FlagState: trimPtr(input.FlagState), Timezone: timezone,
	})
}

func (s *service) GetShip(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.Ship, error) {
	return s.getShipScoped(ctx, p, id)
}

func (s *service) ListShips(ctx context.Context, p auth.Principal, rawCursor string, limit int32) (Page[sqlc.Ship], error) {
	tenantID := p.TenantID
	if p.ActorType == auth.ActorSystem {
		tenantID = uuid.Nil
	}
	parts := []string{"ships", tenantID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[sqlc.Ship]{}, err
	}
	rows, err := s.repo.listShips(ctx, sqlc.ListShipsParams{
		TenantID: nullUUID(tenantID), After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[sqlc.Ship]{}, err
	}
	page := Page[sqlc.Ship]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateShip(ctx context.Context, p auth.Principal, id uuid.UUID, input ShipUpdate) (sqlc.Ship, error) {
	current, err := s.getShipScoped(ctx, p, id)
	if err != nil {
		return sqlc.Ship{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return sqlc.Ship{}, validation("Ship name must not be empty.")
		}
		current.Name = name
	}
	if input.Timezone != nil {
		if err := validateTimezone(*input.Timezone); err != nil {
			return sqlc.Ship{}, err
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
			return sqlc.Ship{}, validation("Ship status is not recognised.")
		}
		current.Status = *input.Status
	}
	return s.repo.updateShip(ctx, sqlc.UpdateShipParams{
		ID: current.ID, Name: current.Name, ImoNumber: current.ImoNumber,
		FlagState: current.FlagState, Timezone: current.Timezone, Status: current.Status,
	})
}

func (s *service) DeleteShip(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.getShipScoped(ctx, p, id); err != nil {
		return err
	}
	return s.repo.deleteShip(ctx, id)
}

func (s *service) CreateDeck(ctx context.Context, p auth.Principal, shipID uuid.UUID, input DeckInput) (sqlc.Deck, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return sqlc.Deck{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || input.Level < 0 {
		return sqlc.Deck{}, validation("Deck name is required and level must not be negative.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.Deck{}, apperrors.Internal(err)
	}
	return s.repo.createDeck(ctx, sqlc.CreateDeckParams{ID: id, ShipID: shipID, Level: input.Level, Name: name})
}

func (s *service) GetDeck(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.Deck, error) {
	deck, err := s.repo.getDeck(ctx, id)
	if err != nil {
		return sqlc.Deck{}, err
	}
	if _, err := s.getShipScoped(ctx, p, deck.ShipID); err != nil {
		return sqlc.Deck{}, err
	}
	return deck, nil
}

func (s *service) ListDecks(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[sqlc.Deck], error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return Page[sqlc.Deck]{}, err
	}
	parts := []string{"decks", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[sqlc.Deck]{}, err
	}
	rows, err := s.repo.listDecks(ctx, sqlc.ListDecksParams{ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[sqlc.Deck]{}, err
	}
	page := Page[sqlc.Deck]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateDeck(ctx context.Context, p auth.Principal, id uuid.UUID, input DeckUpdate) (sqlc.Deck, error) {
	current, err := s.GetDeck(ctx, p, id)
	if err != nil {
		return sqlc.Deck{}, err
	}
	if input.Level != nil {
		if *input.Level < 0 {
			return sqlc.Deck{}, validation("Deck level must not be negative.")
		}
		current.Level = *input.Level
	}
	if input.Name != nil {
		if strings.TrimSpace(*input.Name) == "" {
			return sqlc.Deck{}, validation("Deck name must not be empty.")
		}
		current.Name = strings.TrimSpace(*input.Name)
	}
	return s.repo.updateDeck(ctx, sqlc.UpdateDeckParams{ID: id, Level: current.Level, Name: current.Name})
}

func (s *service) DeleteDeck(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetDeck(ctx, p, id); err != nil {
		return err
	}
	return s.repo.deleteDeck(ctx, id)
}

func (s *service) CreateArea(ctx context.Context, p auth.Principal, shipID uuid.UUID, input AreaInput) (sqlc.Area, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return sqlc.Area{}, err
	}
	if err := s.requireDeck(ctx, shipID, input.DeckID); err != nil {
		return sqlc.Area{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || !areaTypes[input.AreaType] {
		return sqlc.Area{}, validation("Area name and a recognised area type are required.")
	}
	if input.Capacity != nil && *input.Capacity < 0 {
		return sqlc.Area{}, validation("Area capacity must not be negative.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.Area{}, apperrors.Internal(err)
	}
	return s.repo.createArea(ctx, sqlc.CreateAreaParams{
		ID: id, ShipID: shipID, DeckID: input.DeckID, Name: name,
		AreaType: input.AreaType, Capacity: input.Capacity, Layout: input.Layout,
	})
}

func (s *service) GetArea(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.Area, error) {
	area, err := s.repo.getArea(ctx, id)
	if err != nil {
		return sqlc.Area{}, err
	}
	if _, err := s.getShipScoped(ctx, p, area.ShipID); err != nil {
		return sqlc.Area{}, err
	}
	return area, nil
}

func (s *service) ListAreas(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[sqlc.Area], error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return Page[sqlc.Area]{}, err
	}
	parts := []string{"areas", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[sqlc.Area]{}, err
	}
	rows, err := s.repo.listAreas(ctx, sqlc.ListAreasParams{ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[sqlc.Area]{}, err
	}
	page := Page[sqlc.Area]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateArea(ctx context.Context, p auth.Principal, id uuid.UUID, input AreaUpdate) (sqlc.Area, error) {
	current, err := s.GetArea(ctx, p, id)
	if err != nil {
		return sqlc.Area{}, err
	}
	if input.Name != nil {
		if strings.TrimSpace(*input.Name) == "" {
			return sqlc.Area{}, validation("Area name must not be empty.")
		}
		current.Name = strings.TrimSpace(*input.Name)
	}
	if input.Capacity != nil {
		if *input.Capacity < 0 {
			return sqlc.Area{}, validation("Area capacity must not be negative.")
		}
		current.Capacity = input.Capacity
	}
	if input.Layout != nil {
		current.Layout = input.Layout
	}
	return s.repo.updateArea(ctx, sqlc.UpdateAreaParams{
		ID: id, DeckID: current.DeckID, Name: current.Name, AreaType: current.AreaType,
		Capacity: current.Capacity, Layout: current.Layout,
	})
}

func (s *service) DeleteArea(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetArea(ctx, p, id); err != nil {
		return err
	}
	return s.repo.deleteArea(ctx, id)
}

func (s *service) CreateRoom(ctx context.Context, p auth.Principal, shipID uuid.UUID, input RoomInput) (sqlc.Room, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return sqlc.Room{}, err
	}
	if err := s.requireDeck(ctx, shipID, input.DeckID); err != nil {
		return sqlc.Room{}, err
	}
	number := strings.TrimSpace(input.Number)
	if number == "" || !roomTypes[input.RoomType] || input.Capacity <= 0 {
		return sqlc.Room{}, validation("Room number, a recognised room type, and positive capacity are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.Room{}, apperrors.Internal(err)
	}
	return s.repo.createRoom(ctx, sqlc.CreateRoomParams{
		ID: id, ShipID: shipID, DeckID: input.DeckID, Number: number, RoomType: input.RoomType,
		Capacity: input.Capacity, IsBookable: input.IsBookable, Layout: input.Layout,
	})
}

func (s *service) GetRoom(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.Room, error) {
	room, err := s.repo.getRoom(ctx, id)
	if err != nil {
		return sqlc.Room{}, err
	}
	if _, err := s.getShipScoped(ctx, p, room.ShipID); err != nil {
		return sqlc.Room{}, err
	}
	return room, nil
}

func (s *service) ListRooms(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[sqlc.Room], error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return Page[sqlc.Room]{}, err
	}
	parts := []string{"rooms", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[sqlc.Room]{}, err
	}
	rows, err := s.repo.listRooms(ctx, sqlc.ListRoomsParams{ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[sqlc.Room]{}, err
	}
	page := Page[sqlc.Room]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateRoom(ctx context.Context, p auth.Principal, id uuid.UUID, input RoomUpdate) (sqlc.Room, error) {
	current, err := s.GetRoom(ctx, p, id)
	if err != nil {
		return sqlc.Room{}, err
	}
	if input.DeckID != nil {
		if err := s.requireDeck(ctx, current.ShipID, *input.DeckID); err != nil {
			return sqlc.Room{}, err
		}
		current.DeckID = *input.DeckID
	}
	if input.Number != nil {
		if strings.TrimSpace(*input.Number) == "" {
			return sqlc.Room{}, validation("Room number must not be empty.")
		}
		current.Number = strings.TrimSpace(*input.Number)
	}
	if input.RoomType != nil {
		if !roomTypes[*input.RoomType] {
			return sqlc.Room{}, validation("Room type is not recognised.")
		}
		current.RoomType = *input.RoomType
	}
	if input.Capacity != nil {
		if *input.Capacity <= 0 {
			return sqlc.Room{}, validation("Room capacity must be positive.")
		}
		current.Capacity = *input.Capacity
	}
	if input.IsBookable != nil {
		current.IsBookable = *input.IsBookable
	}
	if input.Layout != nil {
		current.Layout = input.Layout
	}
	return s.repo.updateRoom(ctx, sqlc.UpdateRoomParams{
		ID: id, DeckID: current.DeckID, Number: current.Number, RoomType: current.RoomType,
		Capacity: current.Capacity, IsBookable: current.IsBookable, Layout: current.Layout,
	})
}

func (s *service) DeleteRoom(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetRoom(ctx, p, id); err != nil {
		return err
	}
	return s.repo.deleteRoom(ctx, id)
}

func (s *service) Exists(ctx context.Context, shipID uuid.UUID) (bool, error) {
	return s.repo.shipExists(ctx, shipID, uuid.Nil)
}

func (s *service) TenantID(ctx context.Context, shipID uuid.UUID) (uuid.UUID, error) {
	row, err := s.repo.getShip(ctx, shipID)
	if err != nil {
		return uuid.Nil, err
	}
	return row.TenantID, nil
}

func (s *service) RoomShipID(ctx context.Context, roomID uuid.UUID) (uuid.UUID, error) {
	row, err := s.repo.getRoom(ctx, roomID)
	if err != nil {
		return uuid.Nil, err
	}
	return row.ShipID, nil
}

func (s *service) getShipScoped(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.Ship, error) {
	row, err := s.repo.getShip(ctx, id)
	if err != nil {
		return sqlc.Ship{}, err
	}
	if !p.CanAccessTenant(row.TenantID) || !p.CanAccessShip(row.ID) {
		return sqlc.Ship{}, apperrors.Forbidden("Ship access is not permitted.")
	}
	return row, nil
}

func (s *service) requireDeck(ctx context.Context, shipID, deckID uuid.UUID) error {
	ok, err := s.repo.deckExistsInShip(ctx, deckID, shipID)
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

func execResult(n int64, err error) error {
	if err != nil {
		return database.MapPgError(err)
	}
	if n == 0 {
		return database.MapPgError(pgx.ErrNoRows)
	}
	return nil
}
