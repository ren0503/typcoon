package ai

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	CreateAIParam(context.Context, CreateAIParamParams) (AiParam, error)
	GetAIParam(context.Context, uuid.UUID) (AiParam, error)
	ListAIParams(context.Context, ListAIParamsParams) ([]AiParam, error)
	UpdateAIParamMetadata(context.Context, UpdateAIParamMetadataParams) (AiParam, error)
	SoftDeleteAIParam(context.Context, uuid.UUID) error
}
