package tenant

import (
	"context"

	"github.com/google/uuid"
)

// Repository is the persistence port for the tenant module.
type Repository interface {
	Create(context.Context, CreateParams) (Tenant, error)
	Get(context.Context, uuid.UUID) (Tenant, error)
	List(context.Context, ListParams) ([]Tenant, error)
	Update(context.Context, UpdateParams) (Tenant, error)
	SoftDelete(context.Context, uuid.UUID) error
}
