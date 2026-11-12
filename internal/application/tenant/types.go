package tenant

import (
	"time"

	"github.com/google/uuid"
)

// Tenant is the application row type for the tenants table.
type Tenant struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Version   int64      `json:"version"`
	DeletedAt *time.Time `json:"deleted_at"`
}

// CreateParams are the persistence inputs for creating a tenant.
type CreateParams struct {
	ID   uuid.UUID
	Name string
}

// ListParams are the persistence inputs for listing tenants.
type ListParams struct {
	OnlyID    uuid.NullUUID
	After     uuid.NullUUID
	PageLimit int32
}

// UpdateParams are the persistence inputs for updating a tenant.
type UpdateParams struct {
	ID   uuid.UUID
	Name *string
}
