package voyage

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	CreatePort(context.Context, CreatePortParams) (Port, error)
	GetPort(context.Context, uuid.UUID) (Port, error)
	ListPorts(context.Context, ListPortsParams) ([]Port, error)
	UpdatePort(context.Context, UpdatePortParams) (Port, error)
	DeletePort(context.Context, uuid.UUID) error
	PortExists(context.Context, uuid.UUID) (bool, error)

	CreateVoyage(context.Context, CreateVoyageParams) (Voyage, error)
	GetVoyage(context.Context, uuid.UUID) (Voyage, error)
	ListVoyages(context.Context, ListVoyagesParams) ([]Voyage, error)
	ListVoyagesForTenant(context.Context, ListVoyagesForTenantParams) ([]Voyage, error)
	UpdateVoyage(context.Context, UpdateVoyageParams) (Voyage, error)
	DeleteVoyage(context.Context, uuid.UUID) error

	CreatePortCall(context.Context, CreatePortCallParams) (PortCall, error)
	GetPortCall(context.Context, uuid.UUID) (PortCall, error)
	ListPortCalls(context.Context, ListPortCallsParams) ([]PortCall, error)
	UpdatePortCall(context.Context, UpdatePortCallParams) (PortCall, error)
	DeletePortCall(context.Context, uuid.UUID) error
}
