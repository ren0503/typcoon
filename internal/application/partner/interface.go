package partner

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	CreatePartner(context.Context, CreatePartnerParams) (Partner, error)
	GetPartner(context.Context, uuid.UUID) (Partner, error)
	ListPartners(context.Context, ListPartnersParams) ([]Partner, error)
	UpdatePartner(context.Context, UpdatePartnerParams) (Partner, error)
	DeletePartner(context.Context, uuid.UUID) error
	PartnerExists(context.Context, PartnerExistsParams) (bool, error)
	PartnerServiceExists(context.Context, uuid.UUID) (bool, error)
	CreateContract(context.Context, CreatePartnerContractParams) (PartnerContract, error)
	GetContract(context.Context, uuid.UUID) (PartnerContract, error)
	ListContracts(context.Context, ListPartnerContractsParams) ([]PartnerContract, error)
	UpdateContract(context.Context, UpdatePartnerContractParams) (PartnerContract, error)
	DeleteContract(context.Context, uuid.UUID) error
	CreateStaff(context.Context, CreatePartnerStaffParams) (PartnerStaff, error)
	GetStaff(context.Context, uuid.UUID) (PartnerStaff, error)
	ListStaff(context.Context, ListPartnerStaffParams) ([]PartnerStaff, error)
	UpdateStaff(context.Context, UpdatePartnerStaffParams) (PartnerStaff, error)
	DeleteStaff(context.Context, uuid.UUID) error
	CreateService(context.Context, CreatePartnerServiceParams) (PartnerService, error)
	GetService(context.Context, uuid.UUID) (PartnerService, error)
	ListServices(context.Context, ListPartnerServicesParams) ([]PartnerService, error)
	UpdateService(context.Context, UpdatePartnerServiceParams) (PartnerService, error)
	DeleteService(context.Context, uuid.UUID) error
	CreateSlot(context.Context, CreatePartnerSlotParams) (PartnerSlot, error)
	GetSlot(context.Context, uuid.UUID) (PartnerSlot, error)
	ListSlots(context.Context, ListPartnerSlotsParams) ([]PartnerSlot, error)
	UpdateSlot(context.Context, UpdatePartnerSlotParams) (PartnerSlot, error)
	DeleteSlot(context.Context, uuid.UUID) error
}
