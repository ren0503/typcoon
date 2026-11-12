// Package guest implements guest persistence.
package guest

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	appguest "github.com/ren0503/tycoon/internal/application/guest"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Repo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool, q: sqlc.New(pool)} }

var _ appguest.Repository = (*Repo)(nil)

func (r *Repo) Begin(ctx context.Context) (appguest.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, postgres.MapPgError(err)
	}
	return &Tx{tx: tx, q: sqlc.New(tx)}, nil
}
func (r *Repo) GetGuest(ctx context.Context, id uuid.UUID) (appguest.Guest, error) {
	row, err := r.q.GetGuest(ctx, id)
	return convert.To[appguest.Guest](row), postgres.MapPgError(err)
}
func (r *Repo) ListGuests(ctx context.Context, p appguest.ListGuestsParams) ([]appguest.Guest, error) {
	rows, err := r.q.ListGuests(ctx, convert.To[sqlc.ListGuestsParams](p))
	return convert.Slice[appguest.Guest](rows), postgres.MapPgError(err)
}
func (r *Repo) GetBoardingPass(ctx context.Context, id uuid.UUID) (appguest.BoardingPass, error) {
	row, err := r.q.GetBoardingPass(ctx, id)
	return convert.To[appguest.BoardingPass](row), postgres.MapPgError(err)
}
func (r *Repo) ListBoardingPasses(ctx context.Context, p appguest.ListBoardingPassesParams) ([]appguest.BoardingPass, error) {
	rows, err := r.q.ListBoardingPasses(ctx, convert.To[sqlc.ListBoardingPassesParams](p))
	return convert.Slice[appguest.BoardingPass](rows), postgres.MapPgError(err)
}
