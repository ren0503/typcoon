// Package ai implements AI parameter persistence.
package ai

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appai "github.com/ren0503/tycoon/internal/application/ai"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Repo struct{ q *sqlc.Queries }

func New(pool *pgxpool.Pool) *Repo { return &Repo{q: sqlc.New(pool)} }

var _ appai.Repository = (*Repo)(nil)

func (r *Repo) CreateAIParam(ctx context.Context, p appai.CreateAIParamParams) (appai.AiParam, error) {
	row, err := r.q.CreateAIParam(ctx, convert.To[sqlc.CreateAIParamParams](p))
	return convert.To[appai.AiParam](row), postgres.MapPgError(err)
}
func (r *Repo) GetAIParam(ctx context.Context, id uuid.UUID) (appai.AiParam, error) {
	row, err := r.q.GetAIParam(ctx, id)
	return convert.To[appai.AiParam](row), postgres.MapPgError(err)
}
func (r *Repo) ListAIParams(ctx context.Context, p appai.ListAIParamsParams) ([]appai.AiParam, error) {
	rows, err := r.q.ListAIParams(ctx, convert.To[sqlc.ListAIParamsParams](p))
	return convert.Slice[appai.AiParam](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateAIParamMetadata(ctx context.Context, p appai.UpdateAIParamMetadataParams) (appai.AiParam, error) {
	row, err := r.q.UpdateAIParamMetadata(ctx, convert.To[sqlc.UpdateAIParamMetadataParams](p))
	return convert.To[appai.AiParam](row), postgres.MapPgError(err)
}
func (r *Repo) SoftDeleteAIParam(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteAIParam(ctx, id)
	if err != nil {
		return postgres.MapPgError(err)
	}
	if n == 0 {
		return postgres.MapPgError(pgx.ErrNoRows)
	}
	return nil
}
