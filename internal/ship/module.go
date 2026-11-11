package ship

import (
	"context"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

// Page is a cursor-paginated collection of any ship-module resource.
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// ShipInput is the ship creation payload.
type ShipInput struct {
	Name      string
	ImoNumber *string
	FlagState *string
	Timezone  string
	Status    string
}

// ShipUpdate is the ship update payload.
type ShipUpdate struct {
	Name      *string
	ImoNumber *string
	FlagState *string
	Timezone  *string
	Status    *string
}

// DeckInput is the deck creation payload.
type DeckInput struct {
	Level int32
	Name  string
}

// DeckUpdate is the deck update payload.
type DeckUpdate struct {
	Level *int32
	Name  *string
}

// AreaInput is the area creation payload.
type AreaInput struct {
	DeckID   uuid.UUID
	Name     string
	AreaType string
	Capacity *int32
	Layout   []byte
}

// AreaUpdate is the area update payload.
type AreaUpdate struct {
	Name     *string
	Capacity *int32
	Layout   []byte
}

// RoomInput is the room creation payload.
type RoomInput struct {
	DeckID     uuid.UUID
	Number     string
	RoomType   string
	Capacity   int16
	IsBookable bool
	Layout     []byte
}

// RoomUpdate is the room update payload.
type RoomUpdate struct {
	DeckID     *uuid.UUID
	Number     *string
	RoomType   *string
	Capacity   *int16
	IsBookable *bool
	Layout     []byte
}

// Service is the public surface of the ship module.
type Service interface {
	CreateShip(ctx context.Context, principal auth.Principal, input ShipInput) (sqlc.Ship, error)
	GetShip(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.Ship, error)
	ListShips(ctx context.Context, principal auth.Principal, cursor string, limit int32) (Page[sqlc.Ship], error)
	UpdateShip(ctx context.Context, principal auth.Principal, id uuid.UUID, input ShipUpdate) (sqlc.Ship, error)
	DeleteShip(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateDeck(ctx context.Context, principal auth.Principal, shipID uuid.UUID, input DeckInput) (sqlc.Deck, error)
	GetDeck(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.Deck, error)
	ListDecks(ctx context.Context, principal auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[sqlc.Deck], error)
	UpdateDeck(ctx context.Context, principal auth.Principal, id uuid.UUID, input DeckUpdate) (sqlc.Deck, error)
	DeleteDeck(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateArea(ctx context.Context, principal auth.Principal, shipID uuid.UUID, input AreaInput) (sqlc.Area, error)
	GetArea(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.Area, error)
	ListAreas(ctx context.Context, principal auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[sqlc.Area], error)
	UpdateArea(ctx context.Context, principal auth.Principal, id uuid.UUID, input AreaUpdate) (sqlc.Area, error)
	DeleteArea(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateRoom(ctx context.Context, principal auth.Principal, shipID uuid.UUID, input RoomInput) (sqlc.Room, error)
	GetRoom(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.Room, error)
	ListRooms(ctx context.Context, principal auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[sqlc.Room], error)
	UpdateRoom(ctx context.Context, principal auth.Principal, id uuid.UUID, input RoomUpdate) (sqlc.Room, error)
	DeleteRoom(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	// Exists reports whether a non-deleted ship exists.
	Exists(ctx context.Context, shipID uuid.UUID) (bool, error)
	// TenantID returns the owning tenant of a non-deleted ship.
	TenantID(ctx context.Context, shipID uuid.UUID) (uuid.UUID, error)
	// RoomShipID returns the ship a non-deleted room belongs to.
	RoomShipID(ctx context.Context, roomID uuid.UUID) (uuid.UUID, error)

	UploadStructureImport(ctx context.Context, principal auth.Principal, shipID uuid.UUID, fileName string, data []byte, idempotencyKey string) (ImportPreview, error)
	GetStructureImport(ctx context.Context, principal auth.Principal, shipID, importID uuid.UUID) (ShipStructureImportResponse, error)
	PreviewStructureImport(ctx context.Context, principal auth.Principal, shipID, importID uuid.UUID) (ImportPreview, error)
	ConfirmStructureImport(ctx context.Context, principal auth.Principal, shipID, importID uuid.UUID) (ShipStructureImportResponse, error)
	StructureImportTemplate(ctx context.Context) ([]byte, error)
}
