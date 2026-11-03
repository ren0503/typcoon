package guest

import (
	"context"

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

func (r *repo) begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, database.MapPgError(err)
	}
	return tx, nil
}

func (r *repo) getGuest(ctx context.Context, id uuid.UUID) (sqlc.Guest, error) {
	row, err := r.q.GetGuest(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listGuests(ctx context.Context, p sqlc.ListGuestsParams) ([]sqlc.Guest, error) {
	rows, err := r.q.ListGuests(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) getBoardingPass(ctx context.Context, id uuid.UUID) (sqlc.BoardingPass, error) {
	row, err := r.q.GetBoardingPass(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listBoardingPasses(ctx context.Context, p sqlc.ListBoardingPassesParams) ([]sqlc.BoardingPass, error) {
	rows, err := r.q.ListBoardingPasses(ctx, p)
	return rows, database.MapPgError(err)
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}

func optionalUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil || *id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}
