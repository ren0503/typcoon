// Package booking implements booking persistence.
package booking

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	appbooking "github.com/ren0503/tycoon/internal/application/booking"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Repo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool, q: sqlc.New(pool)} }

var _ appbooking.Repository = (*Repo)(nil)

func (r *Repo) Begin(ctx context.Context) (appbooking.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, postgres.MapPgError(err)
	}
	return &Tx{tx: tx, q: sqlc.New(tx)}, nil
}
func (r *Repo) GetBooking(ctx context.Context, id uuid.UUID) (appbooking.Booking, error) {
	row, err := r.q.GetBooking(ctx, id)
	return convert.To[appbooking.Booking](row), postgres.MapPgError(err)
}
func (r *Repo) ListBookings(ctx context.Context, p appbooking.ListBookingsParams) ([]appbooking.Booking, error) {
	rows, err := r.q.ListBookings(ctx, convert.To[sqlc.ListBookingsParams](p))
	return convert.Slice[appbooking.Booking](rows), postgres.MapPgError(err)
}
func (r *Repo) ListBookingsForTenant(ctx context.Context, p appbooking.ListBookingsForTenantParams) ([]appbooking.Booking, error) {
	rows, err := r.q.ListBookingsForTenant(ctx, convert.To[sqlc.ListBookingsForTenantParams](p))
	return convert.Slice[appbooking.Booking](rows), postgres.MapPgError(err)
}
func (r *Repo) BookingExists(ctx context.Context, id uuid.UUID) (bool, error) {
	ok, err := r.q.BookingExists(ctx, id)
	return ok, postgres.MapPgError(err)
}
func (r *Repo) GetServiceBooking(ctx context.Context, id uuid.UUID) (appbooking.ServiceBooking, error) {
	row, err := r.q.GetServiceBooking(ctx, id)
	return convert.To[appbooking.ServiceBooking](row), postgres.MapPgError(err)
}
func (r *Repo) ListServiceBookings(ctx context.Context, p appbooking.ListServiceBookingsParams) ([]appbooking.ServiceBooking, error) {
	rows, err := r.q.ListServiceBookings(ctx, convert.To[sqlc.ListServiceBookingsParams](p))
	return convert.Slice[appbooking.ServiceBooking](rows), postgres.MapPgError(err)
}
