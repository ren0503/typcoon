// Package ai owns AI parameter metadata. It does not perform signing workflows.
package ai

import (
	"context"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

// AIParamService manages AI parameter metadata only.
type AIParamService interface {
	Create(ctx context.Context, principal auth.Principal, input ParamInput) (view ParamView, err error)
	Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (ParamView, error)
	List(ctx context.Context, principal auth.Principal, tenantID uuid.UUID, kind *string, cursor string, limit int32) (pagination.Page[ParamView], error)
	Update(ctx context.Context, principal auth.Principal, id uuid.UUID, input ParamUpdate) (ParamView, error)
	Delete(ctx context.Context, principal auth.Principal, id uuid.UUID) error
}
