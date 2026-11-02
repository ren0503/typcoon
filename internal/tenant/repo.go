package tenant

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

type repo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func newRepo(pool *pgxpool.Pool) *repo {
	return &repo{pool: pool, q: sqlc.New(pool)}
}

func (r *repo) create(ctx context.Context, id uuid.UUID, name string) (sqlc.Tenant, error) {
	row, err := r.q.CreateTenant(ctx, sqlc.CreateTenantParams{ID: id, Name: name})
	return row, database.MapPgError(err)
}

func (r *repo) get(ctx context.Context, id uuid.UUID) (sqlc.Tenant, error) {
	row, err := r.q.GetTenant(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) list(ctx context.Context, onlyID uuid.UUID, after uuid.UUID, limit int32) ([]sqlc.Tenant, error) {
	rows, err := r.q.ListTenants(ctx, sqlc.ListTenantsParams{
		OnlyID:    nullUUID(onlyID),
		After:     nullUUID(after),
		PageLimit: limit,
	})
	return rows, database.MapPgError(err)
}

func (r *repo) update(ctx context.Context, id uuid.UUID, name *string) (sqlc.Tenant, error) {
	row, err := r.q.UpdateTenant(ctx, sqlc.UpdateTenantParams{ID: id, Name: name})
	return row, database.MapPgError(err)
}

func (r *repo) softDelete(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteTenant(ctx, id)
	if err != nil {
		return database.MapPgError(err)
	}
	if n == 0 {
		return database.MapPgError(pgx.ErrNoRows)
	}
	return nil
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}

var _ = errors.Is
