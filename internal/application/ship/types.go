package ship

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CreateShipParams struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	ImoNumber *string   `json:"imo_number"`
	FlagState *string   `json:"flag_state"`
	Timezone  string    `json:"timezone"`
	Status    string    `json:"status"`
}

type UpdateShipParams struct {
	Name      string    `json:"name"`
	ImoNumber *string   `json:"imo_number"`
	FlagState *string   `json:"flag_state"`
	Timezone  string    `json:"timezone"`
	Status    string    `json:"status"`
	ID        uuid.UUID `json:"id"`
}

type ListShipsParams struct {
	TenantID  uuid.NullUUID `json:"tenant_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateDeckParams struct {
	ID     uuid.UUID `json:"id"`
	ShipID uuid.UUID `json:"ship_id"`
	Level  int32     `json:"level"`
	Name   string    `json:"name"`
}

type UpdateDeckParams struct {
	Level int32     `json:"level"`
	Name  string    `json:"name"`
	ID    uuid.UUID `json:"id"`
}

type ListDecksParams struct {
	ShipID    uuid.UUID     `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateAreaParams struct {
	ID       uuid.UUID `json:"id"`
	ShipID   uuid.UUID `json:"ship_id"`
	DeckID   uuid.UUID `json:"deck_id"`
	Name     string    `json:"name"`
	AreaType string    `json:"area_type"`
	Capacity *int32    `json:"capacity"`
	Layout   []byte    `json:"layout"`
}

type UpdateAreaParams struct {
	DeckID   uuid.UUID `json:"deck_id"`
	Name     string    `json:"name"`
	AreaType string    `json:"area_type"`
	Capacity *int32    `json:"capacity"`
	Layout   []byte    `json:"layout"`
	ID       uuid.UUID `json:"id"`
}

type ListAreasParams struct {
	ShipID    uuid.UUID     `json:"ship_id"`
	DeckID    uuid.NullUUID `json:"deck_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateRoomParams struct {
	ID         uuid.UUID `json:"id"`
	ShipID     uuid.UUID `json:"ship_id"`
	DeckID     uuid.UUID `json:"deck_id"`
	Number     string    `json:"number"`
	RoomType   string    `json:"room_type"`
	Capacity   int16     `json:"capacity"`
	IsBookable bool      `json:"is_bookable"`
	Layout     []byte    `json:"layout"`
}

type UpdateRoomParams struct {
	DeckID     uuid.UUID `json:"deck_id"`
	Number     string    `json:"number"`
	RoomType   string    `json:"room_type"`
	Capacity   int16     `json:"capacity"`
	IsBookable bool      `json:"is_bookable"`
	Layout     []byte    `json:"layout"`
	ID         uuid.UUID `json:"id"`
}

type ListRoomsParams struct {
	ShipID    uuid.UUID     `json:"ship_id"`
	DeckID    uuid.NullUUID `json:"deck_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type ShipExistsParams struct {
	ID       uuid.UUID     `json:"id"`
	TenantID uuid.NullUUID `json:"tenant_id"`
}

type DeckExistsInShipParams struct {
	ID     uuid.UUID `json:"id"`
	ShipID uuid.UUID `json:"ship_id"`
}

type Ship struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	Name      string     `json:"name"`
	ImoNumber *string    `json:"imo_number"`
	FlagState *string    `json:"flag_state"`
	Timezone  string     `json:"timezone"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Version   int64      `json:"version"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type Deck struct {
	ID        uuid.UUID  `json:"id"`
	ShipID    uuid.UUID  `json:"ship_id"`
	Level     int32      `json:"level"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Version   int64      `json:"version"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type Area struct {
	ID        uuid.UUID  `json:"id"`
	ShipID    uuid.UUID  `json:"ship_id"`
	DeckID    uuid.UUID  `json:"deck_id"`
	Name      string     `json:"name"`
	AreaType  string     `json:"area_type"`
	Capacity  *int32     `json:"capacity"`
	Layout    []byte     `json:"layout"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Version   int64      `json:"version"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type Room struct {
	ID         uuid.UUID  `json:"id"`
	ShipID     uuid.UUID  `json:"ship_id"`
	DeckID     uuid.UUID  `json:"deck_id"`
	Number     string     `json:"number"`
	RoomType   string     `json:"room_type"`
	Capacity   int16      `json:"capacity"`
	IsBookable bool       `json:"is_bookable"`
	Layout     []byte     `json:"layout"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Version    int64      `json:"version"`
	DeletedAt  *time.Time `json:"deleted_at"`
}

type ShipStructureImport struct {
	ID               uuid.UUID       `json:"id"`
	ShipID           uuid.UUID       `json:"ship_id"`
	IdempotencyKey   *string         `json:"idempotency_key"`
	Status           string          `json:"status"`
	FileName         string          `json:"file_name"`
	FileSizeBytes    int64           `json:"file_size_bytes"`
	FileSha256       string          `json:"file_sha256"`
	DecksTotal       int32           `json:"decks_total"`
	DecksValid       int32           `json:"decks_valid"`
	DecksInvalid     int32           `json:"decks_invalid"`
	AreasTotal       int32           `json:"areas_total"`
	AreasValid       int32           `json:"areas_valid"`
	AreasInvalid     int32           `json:"areas_invalid"`
	RoomsTotal       int32           `json:"rooms_total"`
	RoomsValid       int32           `json:"rooms_valid"`
	RoomsInvalid     int32           `json:"rooms_invalid"`
	ValidatedPayload []byte          `json:"validated_payload"`
	ValidationErrors json.RawMessage `json:"validation_errors"`
	ErrorMessage     *string         `json:"error_message"`
	CreatedBy        *string         `json:"created_by"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	CompletedAt      *time.Time      `json:"completed_at"`
}

type CreateShipStructureImportParams struct {
	ID               uuid.UUID       `json:"id"`
	ShipID           uuid.UUID       `json:"ship_id"`
	IdempotencyKey   *string         `json:"idempotency_key"`
	Status           string          `json:"status"`
	FileName         string          `json:"file_name"`
	FileSizeBytes    int64           `json:"file_size_bytes"`
	FileSha256       string          `json:"file_sha256"`
	DecksTotal       int32           `json:"decks_total"`
	DecksValid       int32           `json:"decks_valid"`
	DecksInvalid     int32           `json:"decks_invalid"`
	AreasTotal       int32           `json:"areas_total"`
	AreasValid       int32           `json:"areas_valid"`
	AreasInvalid     int32           `json:"areas_invalid"`
	RoomsTotal       int32           `json:"rooms_total"`
	RoomsValid       int32           `json:"rooms_valid"`
	RoomsInvalid     int32           `json:"rooms_invalid"`
	ValidatedPayload []byte          `json:"validated_payload"`
	ValidationErrors json.RawMessage `json:"validation_errors"`
	ErrorMessage     *string         `json:"error_message"`
	CreatedBy        *string         `json:"created_by"`
}

type UpdateShipStructureImportValidationParams struct {
	Status           string          `json:"status"`
	DecksTotal       int32           `json:"decks_total"`
	DecksValid       int32           `json:"decks_valid"`
	DecksInvalid     int32           `json:"decks_invalid"`
	AreasTotal       int32           `json:"areas_total"`
	AreasValid       int32           `json:"areas_valid"`
	AreasInvalid     int32           `json:"areas_invalid"`
	RoomsTotal       int32           `json:"rooms_total"`
	RoomsValid       int32           `json:"rooms_valid"`
	RoomsInvalid     int32           `json:"rooms_invalid"`
	ValidatedPayload []byte          `json:"validated_payload"`
	ValidationErrors json.RawMessage `json:"validation_errors"`
	ErrorMessage     *string         `json:"error_message"`
	ID               uuid.UUID       `json:"id"`
	ShipID           uuid.UUID       `json:"ship_id"`
}

type GetShipStructureImportParams struct {
	ID     uuid.UUID `json:"id"`
	ShipID uuid.UUID `json:"ship_id"`
}

type GetShipStructureImportByIdempotencyKeyParams struct {
	ShipID         uuid.UUID `json:"ship_id"`
	IdempotencyKey *string   `json:"idempotency_key"`
}

type GetShipStructureImportForUpdateParams struct {
	ID     uuid.UUID `json:"id"`
	ShipID uuid.UUID `json:"ship_id"`
}

type FailShipStructureImportParams struct {
	ErrorMessage *string   `json:"error_message"`
	ID           uuid.UUID `json:"id"`
	ShipID       uuid.UUID `json:"ship_id"`
}

type SetShipStructureImportErrorParams struct {
	ErrorMessage *string   `json:"error_message"`
	ID           uuid.UUID `json:"id"`
	ShipID       uuid.UUID `json:"ship_id"`
}

type CompleteShipStructureImportParams struct {
	ID     uuid.UUID `json:"id"`
	ShipID uuid.UUID `json:"ship_id"`
}

type SetShipStructureImportProcessingParams struct {
	ID     uuid.UUID `json:"id"`
	ShipID uuid.UUID `json:"ship_id"`
}
