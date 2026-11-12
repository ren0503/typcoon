package ship

import (
	"context"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/audit"
)

type Repository interface {
	Begin(context.Context) (Tx, error)

	CreateShip(context.Context, CreateShipParams) (Ship, error)
	GetShip(context.Context, uuid.UUID) (Ship, error)
	ListShips(context.Context, ListShipsParams) ([]Ship, error)
	UpdateShip(context.Context, UpdateShipParams) (Ship, error)
	DeleteShip(context.Context, uuid.UUID) error
	ShipExists(context.Context, uuid.UUID, uuid.UUID) (bool, error)

	CreateDeck(context.Context, CreateDeckParams) (Deck, error)
	GetDeck(context.Context, uuid.UUID) (Deck, error)
	ListDecks(context.Context, ListDecksParams) ([]Deck, error)
	UpdateDeck(context.Context, UpdateDeckParams) (Deck, error)
	DeleteDeck(context.Context, uuid.UUID) error
	DeckExistsInShip(context.Context, uuid.UUID, uuid.UUID) (bool, error)

	CreateArea(context.Context, CreateAreaParams) (Area, error)
	GetArea(context.Context, uuid.UUID) (Area, error)
	ListAreas(context.Context, ListAreasParams) ([]Area, error)
	UpdateArea(context.Context, UpdateAreaParams) (Area, error)
	DeleteArea(context.Context, uuid.UUID) error

	CreateRoom(context.Context, CreateRoomParams) (Room, error)
	GetRoom(context.Context, uuid.UUID) (Room, error)
	ListRooms(context.Context, ListRoomsParams) ([]Room, error)
	UpdateRoom(context.Context, UpdateRoomParams) (Room, error)
	DeleteRoom(context.Context, uuid.UUID) error

	CreateShipStructureImport(context.Context, CreateShipStructureImportParams) (ShipStructureImport, error)
	GetShipStructureImport(context.Context, GetShipStructureImportParams) (ShipStructureImport, error)
	GetShipStructureImportByIdempotencyKey(context.Context, GetShipStructureImportByIdempotencyKeyParams) (ShipStructureImport, error)
	UpdateShipStructureImportValidation(context.Context, UpdateShipStructureImportValidationParams) (ShipStructureImport, error)
	FailShipStructureImport(context.Context, FailShipStructureImportParams) (ShipStructureImport, error)
	SetShipStructureImportError(context.Context, SetShipStructureImportErrorParams) error
	ExistingDeckLevels(context.Context, uuid.UUID) (map[int32]bool, error)
	ExistingRoomNumbers(context.Context, uuid.UUID) (map[string]bool, error)
}

// Tx is the application-owned transaction for import confirmation.
type Tx interface {
	ShipExists(context.Context, ShipExistsParams) (bool, error)
	GetShipStructureImportForUpdate(context.Context, GetShipStructureImportForUpdateParams) (ShipStructureImport, error)
	FailShipStructureImport(context.Context, FailShipStructureImportParams) (ShipStructureImport, error)
	SetShipStructureImportProcessing(context.Context, SetShipStructureImportProcessingParams) (int64, error)
	CreateDeck(context.Context, CreateDeckParams) (Deck, error)
	CreateArea(context.Context, CreateAreaParams) (Area, error)
	CreateRoom(context.Context, CreateRoomParams) (Room, error)
	CompleteShipStructureImport(context.Context, CompleteShipStructureImportParams) (ShipStructureImport, error)
	ListExistingDeckLevels(context.Context, uuid.UUID) ([]int32, error)
	ListExistingRoomNumbers(context.Context, uuid.UUID) ([]string, error)
	AppendAudit(context.Context, audit.Entry) error
	Commit(context.Context) error
	Rollback(context.Context) error
}
