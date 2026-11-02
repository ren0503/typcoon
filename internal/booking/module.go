package booking

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

// Page is a cursor-paginated collection.
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// BookingInput is the booking creation payload.
type BookingInput struct {
	ShipID          uuid.UUID
	VoyageID        uuid.UUID
	AccountID       *uuid.UUID
	RoomID          *uuid.UUID
	EmbarkSeq       int32
	DisembarkSeq    int32
	ExpiresAt       *time.Time
	SpecialRequests *string
	IdempotencyKey  string
}

// BookingUpdate is the booking update payload. Booking status is immutable.
type BookingUpdate struct {
	RoomID          *uuid.UUID
	EmbarkSeq       *int32
	DisembarkSeq    *int32
	ExpiresAt       *time.Time
	SpecialRequests *string
}

// ServiceBookingInput is the service booking creation payload.
type ServiceBookingInput struct {
	BookingID        uuid.UUID
	PartnerServiceID uuid.UUID
	SlotID           *uuid.UUID
	GuestID          *uuid.UUID
	Quantity         int16
	PriceCents       *int64
	IdempotencyKey   string
}

// ServiceBookingUpdate is the service booking update payload.
type ServiceBookingUpdate struct {
	SlotID     *uuid.UUID
	GuestID    *uuid.UUID
	Quantity   *int16
	Status     *string
	PriceCents *int64
}

// Service is the public surface of the booking module.
type Service interface {
	CreateBooking(ctx context.Context, principal auth.Principal, input BookingInput) (sqlc.Booking, error)
	GetBooking(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.Booking, error)
	ListBookings(ctx context.Context, principal auth.Principal, shipID *uuid.UUID, cursor string, limit int32) (Page[sqlc.Booking], error)
	UpdateBooking(ctx context.Context, principal auth.Principal, id uuid.UUID, input BookingUpdate) (sqlc.Booking, error)
	DeleteBooking(ctx context.Context, principal auth.Principal, id uuid.UUID) error
	BookingExists(ctx context.Context, id uuid.UUID) (bool, error)

	CreateServiceBooking(ctx context.Context, principal auth.Principal, input ServiceBookingInput) (sqlc.ServiceBooking, error)
	GetServiceBooking(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.ServiceBooking, error)
	ListServiceBookings(ctx context.Context, principal auth.Principal, bookingID uuid.UUID, cursor string, limit int32) (Page[sqlc.ServiceBooking], error)
	UpdateServiceBooking(ctx context.Context, principal auth.Principal, id uuid.UUID, input ServiceBookingUpdate) (sqlc.ServiceBooking, error)
	DeleteServiceBooking(ctx context.Context, principal auth.Principal, id uuid.UUID) error
}
