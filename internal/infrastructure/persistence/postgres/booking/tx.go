package booking

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ren0503/tycoon/internal/application/audit"
	appbooking "github.com/ren0503/tycoon/internal/application/booking"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Tx struct {
	tx pgx.Tx
	q  *sqlc.Queries
}

var _ appbooking.Tx = (*Tx)(nil)

func (t *Tx) GetBookingByIdempotency(ctx context.Context, p appbooking.GetBookingByIdempotencyParams) (appbooking.Booking, error) {
	row, err := t.q.GetBookingByIdempotency(ctx, convert.To[sqlc.GetBookingByIdempotencyParams](p))
	return convert.To[appbooking.Booking](row), postgres.MapPgError(err)
}
func (t *Tx) CreateBooking(ctx context.Context, p appbooking.CreateBookingParams) (appbooking.Booking, error) {
	row, err := t.q.CreateBooking(ctx, convert.To[sqlc.CreateBookingParams](p))
	return convert.To[appbooking.Booking](row), postgres.MapPgError(err)
}
func (t *Tx) UpdateBooking(ctx context.Context, p appbooking.UpdateBookingParams) (appbooking.Booking, error) {
	row, err := t.q.UpdateBooking(ctx, convert.To[sqlc.UpdateBookingParams](p))
	return convert.To[appbooking.Booking](row), postgres.MapPgError(err)
}
func (t *Tx) SoftDeleteBooking(ctx context.Context, id uuid.UUID) error {
	return deleteResult(t.q.SoftDeleteBooking(ctx, id))
}
func (t *Tx) GetServiceBookingByIdempotency(ctx context.Context, p appbooking.GetServiceBookingByIdempotencyParams) (appbooking.ServiceBooking, error) {
	row, err := t.q.GetServiceBookingByIdempotency(ctx, convert.To[sqlc.GetServiceBookingByIdempotencyParams](p))
	return convert.To[appbooking.ServiceBooking](row), postgres.MapPgError(err)
}
func (t *Tx) CreateServiceBooking(ctx context.Context, p appbooking.CreateServiceBookingParams) (appbooking.ServiceBooking, error) {
	row, err := t.q.CreateServiceBooking(ctx, convert.To[sqlc.CreateServiceBookingParams](p))
	return convert.To[appbooking.ServiceBooking](row), postgres.MapPgError(err)
}
func (t *Tx) UpdateServiceBooking(ctx context.Context, p appbooking.UpdateServiceBookingParams) (appbooking.ServiceBooking, error) {
	row, err := t.q.UpdateServiceBooking(ctx, convert.To[sqlc.UpdateServiceBookingParams](p))
	return convert.To[appbooking.ServiceBooking](row), postgres.MapPgError(err)
}
func (t *Tx) SoftDeleteServiceBooking(ctx context.Context, id uuid.UUID) error {
	return deleteResult(t.q.SoftDeleteServiceBooking(ctx, id))
}
func (t *Tx) AppendAudit(ctx context.Context, e audit.Entry) error {
	return postgres.AppendAuditTx(ctx, t.tx, e)
}
func (t *Tx) Commit(ctx context.Context) error   { return postgres.MapPgError(t.tx.Commit(ctx)) }
func (t *Tx) Rollback(ctx context.Context) error { return postgres.MapPgError(t.tx.Rollback(ctx)) }
func deleteResult(n int64, err error) error {
	if err != nil {
		return postgres.MapPgError(err)
	}
	if n == 0 {
		return postgres.MapPgError(pgx.ErrNoRows)
	}
	return nil
}
