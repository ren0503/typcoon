package voyage

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

type repo struct {
	q *sqlc.Queries
}

func newRepo(pool *pgxpool.Pool) *repo {
	return &repo{q: sqlc.New(pool)}
}

func (r *repo) createPort(ctx context.Context, p sqlc.CreatePortParams) (sqlc.Port, error) {
	row, err := r.q.CreatePort(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getPort(ctx context.Context, id uuid.UUID) (sqlc.Port, error) {
	row, err := r.q.GetPort(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listPorts(ctx context.Context, after uuid.UUID, limit int32) ([]sqlc.Port, error) {
	rows, err := r.q.ListPorts(ctx, sqlc.ListPortsParams{After: nullUUID(after), PageLimit: limit})
	return rows, database.MapPgError(err)
}

func (r *repo) updatePort(ctx context.Context, p sqlc.UpdatePortParams) (sqlc.Port, error) {
	row, err := r.q.UpdatePort(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deletePort(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeletePort(ctx, id)
	return execResult(n, err)
}

func (r *repo) portExists(ctx context.Context, id uuid.UUID) (bool, error) {
	ok, err := r.q.PortExists(ctx, id)
	return ok, database.MapPgError(err)
}

func (r *repo) createVoyage(ctx context.Context, p sqlc.CreateVoyageParams) (sqlc.Voyage, error) {
	row, err := r.q.CreateVoyage(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getVoyage(ctx context.Context, id uuid.UUID) (sqlc.Voyage, error) {
	row, err := r.q.GetVoyage(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listVoyages(ctx context.Context, p sqlc.ListVoyagesParams) ([]sqlc.Voyage, error) {
	rows, err := r.q.ListVoyages(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) listVoyagesForTenant(ctx context.Context, p sqlc.ListVoyagesForTenantParams) ([]sqlc.Voyage, error) {
	rows, err := r.q.ListVoyagesForTenant(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateVoyage(ctx context.Context, p sqlc.UpdateVoyageParams) (sqlc.Voyage, error) {
	row, err := r.q.UpdateVoyage(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteVoyage(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteVoyage(ctx, id)
	return execResult(n, err)
}

func (r *repo) createPortCall(ctx context.Context, p sqlc.CreatePortCallParams) (sqlc.PortCall, error) {
	row, err := r.q.CreatePortCall(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getPortCall(ctx context.Context, id uuid.UUID) (sqlc.PortCall, error) {
	row, err := r.q.GetPortCall(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listPortCalls(ctx context.Context, p sqlc.ListPortCallsParams) ([]sqlc.PortCall, error) {
	rows, err := r.q.ListPortCalls(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updatePortCall(ctx context.Context, p sqlc.UpdatePortCallParams) (sqlc.PortCall, error) {
	row, err := r.q.UpdatePortCall(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deletePortCall(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeletePortCall(ctx, id)
	return execResult(n, err)
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}
