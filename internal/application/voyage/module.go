package voyage

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
)

// Page is a cursor-paginated collection.
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// PortInput is the port creation payload.
type PortInput struct {
	UnLocode    string
	Name        string
	CountryCode string
	Latitude    *float64
	Longitude   *float64
	Timezone    string
}

// PortUpdate is the port update payload.
type PortUpdate struct {
	Name      *string
	Latitude  *float64
	Longitude *float64
	Timezone  *string
}

// VoyageInput is the voyage creation payload.
type VoyageInput struct {
	ShipID           uuid.UUID
	Code             string
	DepartureAt      time.Time
	ArrivalAt        time.Time
	BoardingOpensAt  *time.Time
	ManifestFrozenAt *time.Time
}

// VoyageUpdate is the voyage update payload.
type VoyageUpdate struct {
	Code             *string
	DepartureAt      *time.Time
	ArrivalAt        *time.Time
	BoardingOpensAt  *time.Time
	ManifestFrozenAt *time.Time
}

// PortCallInput is the port call creation payload.
type PortCallInput struct {
	PortID          uuid.UUID
	Sequence        int32
	Eta             *time.Time
	Etd             *time.Time
	AllowsEmbark    bool
	AllowsDisembark bool
}

// PortCallUpdate is the port call update payload.
type PortCallUpdate struct {
	Eta             *time.Time
	Etd             *time.Time
	Ata             *time.Time
	Atd             *time.Time
	AllowsEmbark    *bool
	AllowsDisembark *bool
	Status          *string
}

// Service is the public surface of the voyage module.
type Service interface {
	CreatePort(ctx context.Context, principal auth.Principal, input PortInput) (Port, error)
	GetPort(ctx context.Context, principal auth.Principal, id uuid.UUID) (Port, error)
	ListPorts(ctx context.Context, principal auth.Principal, cursor string, limit int32) (Page[Port], error)
	UpdatePort(ctx context.Context, principal auth.Principal, id uuid.UUID, input PortUpdate) (Port, error)
	DeletePort(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateVoyage(ctx context.Context, principal auth.Principal, input VoyageInput) (Voyage, error)
	GetVoyage(ctx context.Context, principal auth.Principal, id uuid.UUID) (Voyage, error)
	ListVoyages(ctx context.Context, principal auth.Principal, shipID *uuid.UUID, cursor string, limit int32) (Page[Voyage], error)
	UpdateVoyage(ctx context.Context, principal auth.Principal, id uuid.UUID, input VoyageUpdate) (Voyage, error)
	DeleteVoyage(ctx context.Context, principal auth.Principal, id uuid.UUID) error
	// Lookup returns a voyage without a principal check. Callers must have
	// already established scope through the owning ship.
	Lookup(ctx context.Context, id uuid.UUID) (Voyage, error)

	CreatePortCall(ctx context.Context, principal auth.Principal, voyageID uuid.UUID, input PortCallInput) (PortCall, error)
	GetPortCall(ctx context.Context, principal auth.Principal, id uuid.UUID) (PortCall, error)
	ListPortCalls(ctx context.Context, principal auth.Principal, voyageID uuid.UUID, cursor string, limit int32) (Page[PortCall], error)
	UpdatePortCall(ctx context.Context, principal auth.Principal, id uuid.UUID, input PortCallUpdate) (PortCall, error)
	DeletePortCall(ctx context.Context, principal auth.Principal, id uuid.UUID) error
}
