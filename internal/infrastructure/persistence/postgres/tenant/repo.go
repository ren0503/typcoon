// Package tenant implements tenant persistence.
package tenant

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apptenant "github.com/ren0503/tycoon/internal/application/tenant"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

// Repo implements the tenant application repository port.
type Repo struct {
	q *sqlc.Queries
}

// New builds a tenant repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{q: sqlc.New(pool)}
}

var _ apptenant.Repository = (*Repo)(nil)

func (r *Repo) Create(ctx context.Context, p apptenant.CreateParams) (apptenant.Tenant, error) {
	row, err := r.q.CreateTenant(ctx, sqlc.CreateTenantParams{ID: p.ID, Name: p.Name})
	return convertTenant(row), postgres.MapPgError(err)
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (apptenant.Tenant, error) {
	row, err := r.q.GetTenant(ctx, id)
	return convertTenant(row), postgres.MapPgError(err)
}

func (r *Repo) List(ctx context.Context, p apptenant.ListParams) ([]apptenant.Tenant, error) {
	rows, err := r.q.ListTenants(ctx, sqlc.ListTenantsParams{OnlyID: p.OnlyID, After: p.After, PageLimit: p.PageLimit})
	out := make([]apptenant.Tenant, len(rows))
	for i, row := range rows {
		out[i] = convertTenant(row)
	}
	return out, postgres.MapPgError(err)
}

func (r *Repo) Update(ctx context.Context, p apptenant.UpdateParams) (apptenant.Tenant, error) {
	row, err := r.q.UpdateTenant(ctx, sqlc.UpdateTenantParams{ID: p.ID, Name: p.Name})
	return convertTenant(row), postgres.MapPgError(err)
}

func (r *Repo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteTenant(ctx, id)
	if err != nil {
		return postgres.MapPgError(err)
	}
	if n == 0 {
		return postgres.MapPgError(pgx.ErrNoRows)
	}
	return nil
}

func convertTenant(row sqlc.Tenant) apptenant.Tenant {
	return apptenant.Tenant{
		ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt, Version: row.Version, DeletedAt: row.DeletedAt,
	}
}
