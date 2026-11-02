package booking

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

func (r *repo) createBooking(ctx context.Context, p sqlc.CreateBookingParams) (sqlc.Booking, error) {
	row, err := r.q.CreateBooking(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getBooking(ctx context.Context, id uuid.UUID) (sqlc.Booking, error) {
	row, err := r.q.GetBooking(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) getBookingByIdempotency(ctx context.Context, accountID uuid.NullUUID, key string) (sqlc.Booking, error) {
	row, err := r.q.GetBookingByIdempotency(ctx, sqlc.GetBookingByIdempotencyParams{AccountID: accountID, IdempotencyKey: &key})
	return row, database.MapPgError(err)
}

func (r *repo) listBookings(ctx context.Context, p sqlc.ListBookingsParams) ([]sqlc.Booking, error) {
	rows, err := r.q.ListBookings(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) listBookingsForTenant(ctx context.Context, p sqlc.ListBookingsForTenantParams) ([]sqlc.Booking, error) {
	rows, err := r.q.ListBookingsForTenant(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateBooking(ctx context.Context, p sqlc.UpdateBookingParams) (sqlc.Booking, error) {
	row, err := r.q.UpdateBooking(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteBooking(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteBooking(ctx, id)
	return execResult(n, err)
}

func (r *repo) bookingExists(ctx context.Context, id uuid.UUID) (bool, error) {
	ok, err := r.q.BookingExists(ctx, id)
	return ok, database.MapPgError(err)
}

func (r *repo) createServiceBooking(ctx context.Context, p sqlc.CreateServiceBookingParams) (sqlc.ServiceBooking, error) {
	row, err := r.q.CreateServiceBooking(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getServiceBooking(ctx context.Context, id uuid.UUID) (sqlc.ServiceBooking, error) {
	row, err := r.q.GetServiceBooking(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) getServiceBookingByIdempotency(ctx context.Context, bookingID uuid.UUID, key string) (sqlc.ServiceBooking, error) {
	row, err := r.q.GetServiceBookingByIdempotency(ctx, sqlc.GetServiceBookingByIdempotencyParams{BookingID: bookingID, IdempotencyKey: &key})
	return row, database.MapPgError(err)
}

func (r *repo) listServiceBookings(ctx context.Context, p sqlc.ListServiceBookingsParams) ([]sqlc.ServiceBooking, error) {
	rows, err := r.q.ListServiceBookings(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateServiceBooking(ctx context.Context, p sqlc.UpdateServiceBookingParams) (sqlc.ServiceBooking, error) {
	row, err := r.q.UpdateServiceBooking(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteServiceBooking(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteServiceBooking(ctx, id)
	return execResult(n, err)
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
