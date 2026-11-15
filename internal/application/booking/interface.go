package booking

import (
	"context"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/audit"
)

type Repository interface {
	Begin(context.Context) (Tx, error)
	GetBooking(context.Context, uuid.UUID) (Booking, error)
	ListBookings(context.Context, ListBookingsParams) ([]Booking, error)
	ListBookingsForTenant(context.Context, ListBookingsForTenantParams) ([]Booking, error)
	BookingExists(context.Context, uuid.UUID) (bool, error)
	GetServiceBooking(context.Context, uuid.UUID) (ServiceBooking, error)
	ListServiceBookings(context.Context, ListServiceBookingsParams) ([]ServiceBooking, error)
}

type Tx interface {
	GetBookingByIdempotency(context.Context, GetBookingByIdempotencyParams) (Booking, error)
	CreateBooking(context.Context, CreateBookingParams) (Booking, error)
	UpdateBooking(context.Context, UpdateBookingParams) (Booking, error)
	SoftDeleteBooking(context.Context, uuid.UUID) error
	GetServiceBookingByIdempotency(context.Context, GetServiceBookingByIdempotencyParams) (ServiceBooking, error)
	CreateServiceBooking(context.Context, CreateServiceBookingParams) (ServiceBooking, error)
	UpdateServiceBooking(context.Context, UpdateServiceBookingParams) (ServiceBooking, error)
	SoftDeleteServiceBooking(context.Context, uuid.UUID) error
	AppendAudit(context.Context, audit.Entry) error
	Commit(context.Context) error
	Rollback(context.Context) error
}
