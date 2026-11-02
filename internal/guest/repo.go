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

func (r *repo) createGuest(ctx context.Context, p sqlc.CreateGuestParams) (sqlc.Guest, error) {
	row, err := r.q.CreateGuest(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getGuest(ctx context.Context, id uuid.UUID) (sqlc.Guest, error) {
	row, err := r.q.GetGuest(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listGuests(ctx context.Context, p sqlc.ListGuestsParams) ([]sqlc.Guest, error) {
	rows, err := r.q.ListGuests(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateGuest(ctx context.Context, p sqlc.UpdateGuestParams) (sqlc.Guest, error) {
	row, err := r.q.UpdateGuest(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteGuest(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteGuest(ctx, id)
	return execResult(n, err)
}

func (r *repo) createBoardingPass(ctx context.Context, p sqlc.CreateBoardingPassParams) (sqlc.BoardingPass, error) {
	row, err := r.q.CreateBoardingPass(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getBoardingPass(ctx context.Context, id uuid.UUID) (sqlc.BoardingPass, error) {
	row, err := r.q.GetBoardingPass(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listBoardingPasses(ctx context.Context, p sqlc.ListBoardingPassesParams) ([]sqlc.BoardingPass, error) {
	rows, err := r.q.ListBoardingPasses(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) revokeBoardingPass(ctx context.Context, id uuid.UUID) (sqlc.BoardingPass, error) {
	row, err := r.q.RevokeBoardingPass(ctx, id)
	return row, database.MapPgError(err)
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

func execResult(n int64, err error) error {
	if err != nil {
		return database.MapPgError(err)
	}
	if n == 0 {
		return database.MapPgError(pgx.ErrNoRows)
	}
	return nil
}
