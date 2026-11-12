package partner

import (
	"time"

	"github.com/google/uuid"
)

type Partner struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Name         string     `json:"name"`
	BusinessType string     `json:"business_type"`
	ContactName  *string    `json:"contact_name"`
	ContactEmail *string    `json:"contact_email"`
	ContactPhone *string    `json:"contact_phone"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Version      int64      `json:"version"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

type PartnerContract struct {
	ID              uuid.UUID  `json:"id"`
	ShipID          uuid.UUID  `json:"ship_id"`
	PartnerID       uuid.UUID  `json:"partner_id"`
	StartsOn        *time.Time `json:"starts_on"`
	EndsOn          *time.Time `json:"ends_on"`
	FeeModel        *string    `json:"fee_model"`
	FeeAmountCents  *int64     `json:"fee_amount_cents"`
	Currency        *string    `json:"currency"`
	TermsStorageKey *string    `json:"terms_storage_key"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Version         int64      `json:"version"`
	DeletedAt       *time.Time `json:"deleted_at"`
}

type PartnerStaff struct {
	ID        uuid.UUID  `json:"id"`
	ShipID    uuid.UUID  `json:"ship_id"`
	PartnerID uuid.UUID  `json:"partner_id"`
	FullName  string     `json:"full_name"`
	RoleTitle *string    `json:"role_title"`
	Phone     *string    `json:"phone"`
	Status    string     `json:"status"`
	ValidFrom *time.Time `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Version   int64      `json:"version"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type PartnerService struct {
	ID              uuid.UUID     `json:"id"`
	ShipID          uuid.UUID     `json:"ship_id"`
	PartnerID       uuid.UUID     `json:"partner_id"`
	AreaID          uuid.NullUUID `json:"area_id"`
	Name            string        `json:"name"`
	Category        string        `json:"category"`
	PriceCents      *int64        `json:"price_cents"`
	Currency        *string       `json:"currency"`
	RequiresSlot    bool          `json:"requires_slot"`
	DefaultCapacity *int32        `json:"default_capacity"`
	IsActive        bool          `json:"is_active"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Version         int64         `json:"version"`
	DeletedAt       *time.Time    `json:"deleted_at"`
}

type PartnerSlot struct {
	ID           uuid.UUID     `json:"id"`
	ShipID       uuid.UUID     `json:"ship_id"`
	PartnerID    uuid.UUID     `json:"partner_id"`
	AreaID       uuid.UUID     `json:"area_id"`
	VoyageID     uuid.NullUUID `json:"voyage_id"`
	ServiceID    uuid.NullUUID `json:"service_id"`
	StartAt      time.Time     `json:"start_at"`
	EndAt        time.Time     `json:"end_at"`
	Capacity     *int32        `json:"capacity"`
	Status       string        `json:"status"`
	Source       string        `json:"source"`
	RequestedBy  uuid.NullUUID `json:"requested_by"`
	DecidedBy    uuid.NullUUID `json:"decided_by"`
	DecidedAt    *time.Time    `json:"decided_at"`
	DecisionNote *string       `json:"decision_note"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Version      int64         `json:"version"`
	DeletedAt    *time.Time    `json:"deleted_at"`
}

type CreatePartnerParams struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Name         string    `json:"name"`
	BusinessType string    `json:"business_type"`
	ContactName  *string   `json:"contact_name"`
	ContactEmail *string   `json:"contact_email"`
	ContactPhone *string   `json:"contact_phone"`
}

type UpdatePartnerParams struct {
	Name         string    `json:"name"`
	BusinessType string    `json:"business_type"`
	ContactName  *string   `json:"contact_name"`
	ContactEmail *string   `json:"contact_email"`
	ContactPhone *string   `json:"contact_phone"`
	Status       string    `json:"status"`
	ID           uuid.UUID `json:"id"`
}

type ListPartnersParams struct {
	TenantID  uuid.NullUUID `json:"tenant_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreatePartnerContractParams struct {
	ID              uuid.UUID  `json:"id"`
	ShipID          uuid.UUID  `json:"ship_id"`
	PartnerID       uuid.UUID  `json:"partner_id"`
	StartsOn        *time.Time `json:"starts_on"`
	EndsOn          *time.Time `json:"ends_on"`
	FeeModel        *string    `json:"fee_model"`
	FeeAmountCents  *int64     `json:"fee_amount_cents"`
	Currency        *string    `json:"currency"`
	TermsStorageKey *string    `json:"terms_storage_key"`
	Status          string     `json:"status"`
}

type UpdatePartnerContractParams struct {
	ShipID          uuid.UUID  `json:"ship_id"`
	StartsOn        *time.Time `json:"starts_on"`
	EndsOn          *time.Time `json:"ends_on"`
	FeeModel        *string    `json:"fee_model"`
	FeeAmountCents  *int64     `json:"fee_amount_cents"`
	Currency        *string    `json:"currency"`
	TermsStorageKey *string    `json:"terms_storage_key"`
	Status          string     `json:"status"`
	ID              uuid.UUID  `json:"id"`
}

type ListPartnerContractsParams struct {
	PartnerID uuid.UUID     `json:"partner_id"`
	ShipID    uuid.NullUUID `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreatePartnerStaffParams struct {
	ID        uuid.UUID  `json:"id"`
	ShipID    uuid.UUID  `json:"ship_id"`
	PartnerID uuid.UUID  `json:"partner_id"`
	FullName  string     `json:"full_name"`
	RoleTitle *string    `json:"role_title"`
	Phone     *string    `json:"phone"`
	Status    string     `json:"status"`
	ValidFrom *time.Time `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to"`
}

type UpdatePartnerStaffParams struct {
	ShipID    uuid.UUID  `json:"ship_id"`
	FullName  string     `json:"full_name"`
	RoleTitle *string    `json:"role_title"`
	Phone     *string    `json:"phone"`
	Status    string     `json:"status"`
	ValidFrom *time.Time `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to"`
	ID        uuid.UUID  `json:"id"`
}

type ListPartnerStaffParams struct {
	PartnerID uuid.UUID     `json:"partner_id"`
	ShipID    uuid.NullUUID `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreatePartnerServiceParams struct {
	ID              uuid.UUID     `json:"id"`
	ShipID          uuid.UUID     `json:"ship_id"`
	PartnerID       uuid.UUID     `json:"partner_id"`
	AreaID          uuid.NullUUID `json:"area_id"`
	Name            string        `json:"name"`
	Category        string        `json:"category"`
	PriceCents      *int64        `json:"price_cents"`
	Currency        *string       `json:"currency"`
	RequiresSlot    bool          `json:"requires_slot"`
	DefaultCapacity *int32        `json:"default_capacity"`
	IsActive        bool          `json:"is_active"`
}

type UpdatePartnerServiceParams struct {
	ShipID          uuid.UUID     `json:"ship_id"`
	AreaID          uuid.NullUUID `json:"area_id"`
	Name            string        `json:"name"`
	Category        string        `json:"category"`
	PriceCents      *int64        `json:"price_cents"`
	Currency        *string       `json:"currency"`
	RequiresSlot    bool          `json:"requires_slot"`
	DefaultCapacity *int32        `json:"default_capacity"`
	IsActive        bool          `json:"is_active"`
	ID              uuid.UUID     `json:"id"`
}

type ListPartnerServicesParams struct {
	PartnerID uuid.UUID     `json:"partner_id"`
	ShipID    uuid.NullUUID `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreatePartnerSlotParams struct {
	ID        uuid.UUID     `json:"id"`
	ShipID    uuid.UUID     `json:"ship_id"`
	PartnerID uuid.UUID     `json:"partner_id"`
	AreaID    uuid.UUID     `json:"area_id"`
	VoyageID  uuid.NullUUID `json:"voyage_id"`
	ServiceID uuid.NullUUID `json:"service_id"`
	StartAt   time.Time     `json:"start_at"`
	EndAt     time.Time     `json:"end_at"`
	Capacity  *int32        `json:"capacity"`
}

type UpdatePartnerSlotParams struct {
	VoyageID     uuid.NullUUID `json:"voyage_id"`
	ServiceID    uuid.NullUUID `json:"service_id"`
	StartAt      time.Time     `json:"start_at"`
	EndAt        time.Time     `json:"end_at"`
	Capacity     *int32        `json:"capacity"`
	Status       string        `json:"status"`
	RequestedBy  uuid.NullUUID `json:"requested_by"`
	DecidedBy    uuid.NullUUID `json:"decided_by"`
	DecidedAt    *time.Time    `json:"decided_at"`
	DecisionNote *string       `json:"decision_note"`
	ID           uuid.UUID     `json:"id"`
}

type ListPartnerSlotsParams struct {
	PartnerID uuid.UUID     `json:"partner_id"`
	ShipID    uuid.NullUUID `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type PartnerExistsParams struct {
	ID       uuid.UUID     `json:"id"`
	TenantID uuid.NullUUID `json:"tenant_id"`
}
