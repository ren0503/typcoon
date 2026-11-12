package partner

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

// PartnerInput is the partner creation payload.
type PartnerInput struct {
	Name         string
	BusinessType string
	ContactName  *string
	ContactEmail *string
	ContactPhone *string
}

// PartnerUpdate is the partner update payload.
type PartnerUpdate struct {
	Name         *string
	ContactName  *string
	ContactEmail *string
	ContactPhone *string
	Status       *string
}

// ContractInput is the partner contract creation payload.
type ContractInput struct {
	ShipID          uuid.UUID
	StartsOn        *time.Time
	EndsOn          *time.Time
	FeeModel        *string
	FeeAmountCents  *int64
	Currency        *string
	TermsStorageKey *string
	Status          string
}

// ContractUpdate is the partner contract update payload.
type ContractUpdate struct {
	ShipID          *uuid.UUID
	StartsOn        *time.Time
	EndsOn          *time.Time
	FeeModel        *string
	FeeAmountCents  *int64
	Currency        *string
	TermsStorageKey *string
	Status          *string
}

// StaffInput is the partner staff creation payload.
type StaffInput struct {
	ShipID    uuid.UUID
	FullName  string
	RoleTitle *string
	Phone     *string
	Status    string
	ValidFrom *time.Time
	ValidTo   *time.Time
}

// StaffUpdate is the partner staff update payload.
type StaffUpdate struct {
	ShipID    *uuid.UUID
	FullName  *string
	RoleTitle *string
	Phone     *string
	Status    *string
	ValidFrom *time.Time
	ValidTo   *time.Time
}

// ServiceInput is the partner service creation payload.
type ServiceInput struct {
	ShipID          uuid.UUID
	AreaID          *uuid.UUID
	Name            string
	Category        string
	PriceCents      *int64
	Currency        *string
	RequiresSlot    bool
	DefaultCapacity *int32
	IsActive        bool
}

// ServiceUpdate is the partner service update payload.
type ServiceUpdate struct {
	ShipID          *uuid.UUID
	AreaID          *uuid.UUID
	Name            *string
	Category        *string
	PriceCents      *int64
	Currency        *string
	RequiresSlot    *bool
	DefaultCapacity *int32
	IsActive        *bool
}

// SlotInput is the partner slot creation payload.
type SlotInput struct {
	ShipID    uuid.UUID
	AreaID    uuid.UUID
	VoyageID  *uuid.UUID
	ServiceID *uuid.UUID
	StartAt   time.Time
	EndAt     time.Time
	Capacity  *int32
}

// SlotUpdate is the partner slot decision/update payload.
type SlotUpdate struct {
	VoyageID     *uuid.UUID
	ServiceID    *uuid.UUID
	StartAt      *time.Time
	EndAt        *time.Time
	Capacity     *int32
	Status       *string
	DecisionNote *string
}

// Service is the public surface of the partner module.
type Service interface {
	CreatePartner(ctx context.Context, principal auth.Principal, input PartnerInput) (Partner, error)
	GetPartner(ctx context.Context, principal auth.Principal, id uuid.UUID) (Partner, error)
	ListPartners(ctx context.Context, principal auth.Principal, cursor string, limit int32) (Page[Partner], error)
	UpdatePartner(ctx context.Context, principal auth.Principal, id uuid.UUID, input PartnerUpdate) (Partner, error)
	DeletePartner(ctx context.Context, principal auth.Principal, id uuid.UUID) error
	PartnerExists(ctx context.Context, id uuid.UUID) (bool, error)
	ServiceExists(ctx context.Context, id uuid.UUID) (bool, error)
	SlotExists(ctx context.Context, id uuid.UUID) (bool, error)

	CreateContract(ctx context.Context, principal auth.Principal, partnerID uuid.UUID, input ContractInput) (PartnerContract, error)
	GetContract(ctx context.Context, principal auth.Principal, id uuid.UUID) (PartnerContract, error)
	ListContracts(ctx context.Context, principal auth.Principal, partnerID uuid.UUID, cursor string, limit int32) (Page[PartnerContract], error)
	UpdateContract(ctx context.Context, principal auth.Principal, id uuid.UUID, input ContractUpdate) (PartnerContract, error)
	DeleteContract(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateStaff(ctx context.Context, principal auth.Principal, partnerID uuid.UUID, input StaffInput) (PartnerStaff, error)
	GetStaff(ctx context.Context, principal auth.Principal, id uuid.UUID) (PartnerStaff, error)
	ListStaff(ctx context.Context, principal auth.Principal, partnerID uuid.UUID, cursor string, limit int32) (Page[PartnerStaff], error)
	UpdateStaff(ctx context.Context, principal auth.Principal, id uuid.UUID, input StaffUpdate) (PartnerStaff, error)
	DeleteStaff(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateService(ctx context.Context, principal auth.Principal, partnerID uuid.UUID, input ServiceInput) (PartnerService, error)
	GetService(ctx context.Context, principal auth.Principal, id uuid.UUID) (PartnerService, error)
	ListServices(ctx context.Context, principal auth.Principal, partnerID uuid.UUID, cursor string, limit int32) (Page[PartnerService], error)
	UpdateService(ctx context.Context, principal auth.Principal, id uuid.UUID, input ServiceUpdate) (PartnerService, error)
	DeleteService(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateSlot(ctx context.Context, principal auth.Principal, partnerID uuid.UUID, input SlotInput) (PartnerSlot, error)
	GetSlot(ctx context.Context, principal auth.Principal, id uuid.UUID) (PartnerSlot, error)
	ListSlots(ctx context.Context, principal auth.Principal, partnerID uuid.UUID, cursor string, limit int32) (Page[PartnerSlot], error)
	UpdateSlot(ctx context.Context, principal auth.Principal, id uuid.UUID, input SlotUpdate) (PartnerSlot, error)
	DeleteSlot(ctx context.Context, principal auth.Principal, id uuid.UUID) error
}
