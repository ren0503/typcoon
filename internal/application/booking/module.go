package booking

import (
	"context"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

// Service is the public surface of the booking module.
type Service interface {
	CreateBooking(ctx context.Context, principal auth.Principal, input BookingInput) (Booking, error)
	GetBooking(ctx context.Context, principal auth.Principal, id uuid.UUID) (Booking, error)
	ListBookings(ctx context.Context, principal auth.Principal, shipID *uuid.UUID, cursor string, limit int32) (pagination.Page[Booking], error)
	UpdateBooking(ctx context.Context, principal auth.Principal, id uuid.UUID, input BookingUpdate) (Booking, error)
	DeleteBooking(ctx context.Context, principal auth.Principal, id uuid.UUID) error
	BookingExists(ctx context.Context, id uuid.UUID) (bool, error)

	CreateServiceBooking(ctx context.Context, principal auth.Principal, input ServiceBookingInput) (ServiceBooking, error)
	GetServiceBooking(ctx context.Context, principal auth.Principal, id uuid.UUID) (ServiceBooking, error)
	ListServiceBookings(ctx context.Context, principal auth.Principal, bookingID uuid.UUID, cursor string, limit int32) (pagination.Page[ServiceBooking], error)
	UpdateServiceBooking(ctx context.Context, principal auth.Principal, id uuid.UUID, input ServiceBookingUpdate) (ServiceBooking, error)
	DeleteServiceBooking(ctx context.Context, principal auth.Principal, id uuid.UUID) error
}
