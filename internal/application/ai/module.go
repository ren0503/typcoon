// Package ai owns AI parameter metadata. It does not perform signing workflows.
package ai

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
)

// ParamInput creates an AI parameter metadata record.
type ParamInput struct {
	TenantID     uuid.UUID
	Kind         string
	ParamVersion int32
	StorageKey   string
	Sha256       string
	Kid          string
	Signature    string
	Status       string
	ReleasedAt   *time.Time
	Notes        *string
}

// ParamUpdate updates metadata for an existing AI parameter.
type ParamUpdate struct {
	StorageKey string
	Sha256     string
	Kid        string
	Signature  string
	Status     string
	ReleasedAt *time.Time
	Notes      *string
}

// Page is a cursor-paginated collection.
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// AIParamService manages AI parameter metadata only.
type AIParamService interface {
	Create(ctx context.Context, principal auth.Principal, input ParamInput) (view ParamView, err error)
	Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (ParamView, error)
	List(ctx context.Context, principal auth.Principal, tenantID uuid.UUID, kind *string, cursor string, limit int32) (Page[ParamView], error)
	Update(ctx context.Context, principal auth.Principal, id uuid.UUID, input ParamUpdate) (ParamView, error)
	Delete(ctx context.Context, principal auth.Principal, id uuid.UUID) error
}
