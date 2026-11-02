package tenant

import (
	"context"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/database/sqlc"
	"github.com/ren0503/tycoon/internal/pagination"
)

// Page is a cursor-paginated tenant collection.
type Page struct {
	Items      []sqlc.Tenant
	NextCursor string
}

// CreateInput is the tenant creation payload.
type CreateInput struct {
	Name string
}

// UpdateInput is the tenant update payload.
type UpdateInput struct {
	Name *string
}

// TenantService is the public surface of the tenant module.
type TenantService interface {
	Create(ctx context.Context, principal auth.Principal, input CreateInput) (sqlc.Tenant, error)
	Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.Tenant, error)
	List(ctx context.Context, principal auth.Principal, params pagination.Cursor, limit int32) (Page, error)
	Update(ctx context.Context, principal auth.Principal, id uuid.UUID, input UpdateInput) (sqlc.Tenant, error)
	Delete(ctx context.Context, principal auth.Principal, id uuid.UUID) error
	// GetActive returns a tenant only when the principal may access it. It is
	// used by other modules for tenant validation.
	GetActive(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.Tenant, error)
}
