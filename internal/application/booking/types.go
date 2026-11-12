package booking

import (
	"time"

	"github.com/google/uuid"
)

type Booking struct {
	ID              uuid.UUID     `json:"id"`
	ShipID          uuid.UUID     `json:"ship_id"`
	VoyageID        uuid.UUID     `json:"voyage_id"`
	AccountID       uuid.NullUUID `json:"account_id"`
	RoomID          uuid.NullUUID `json:"room_id"`
	EmbarkSeq       int32         `json:"embark_seq"`
	DisembarkSeq    int32         `json:"disembark_seq"`
	Status          string        `json:"status"`
	ExpiresAt       *time.Time    `json:"expires_at"`
	IdempotencyKey  *string       `json:"idempotency_key"`
	SpecialRequests *string       `json:"special_requests"`
	Source          string        `json:"source"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Version         int64         `json:"version"`
	DeletedAt       *time.Time    `json:"deleted_at"`
}

type ServiceBooking struct {
	ID               uuid.UUID     `json:"id"`
	BookingID        uuid.UUID     `json:"booking_id"`
	PartnerServiceID uuid.UUID     `json:"partner_service_id"`
	SlotID           uuid.NullUUID `json:"slot_id"`
	GuestID          uuid.NullUUID `json:"guest_id"`
	Quantity         int16         `json:"quantity"`
	Status           string        `json:"status"`
	Source           string        `json:"source"`
	IdempotencyKey   *string       `json:"idempotency_key"`
	PriceCents       *int64        `json:"price_cents"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	Version          int64         `json:"version"`
	DeletedAt        *time.Time    `json:"deleted_at"`
}

type CreateBookingParams struct {
	ID              uuid.UUID     `json:"id"`
	ShipID          uuid.UUID     `json:"ship_id"`
	VoyageID        uuid.UUID     `json:"voyage_id"`
	AccountID       uuid.NullUUID `json:"account_id"`
	RoomID          uuid.NullUUID `json:"room_id"`
	EmbarkSeq       int32         `json:"embark_seq"`
	DisembarkSeq    int32         `json:"disembark_seq"`
	ExpiresAt       *time.Time    `json:"expires_at"`
	IdempotencyKey  *string       `json:"idempotency_key"`
	SpecialRequests *string       `json:"special_requests"`
}

type UpdateBookingParams struct {
	RoomID          uuid.NullUUID `json:"room_id"`
	EmbarkSeq       int32         `json:"embark_seq"`
	DisembarkSeq    int32         `json:"disembark_seq"`
	ExpiresAt       *time.Time    `json:"expires_at"`
	SpecialRequests *string       `json:"special_requests"`
	ID              uuid.UUID     `json:"id"`
}

type ListBookingsParams struct {
	ShipID    uuid.NullUUID `json:"ship_id"`
	VoyageID  uuid.NullUUID `json:"voyage_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type ListBookingsForTenantParams struct {
	TenantID  uuid.UUID     `json:"tenant_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type GetBookingByIdempotencyParams struct {
	AccountID      uuid.NullUUID `json:"account_id"`
	IdempotencyKey *string       `json:"idempotency_key"`
}

type CreateServiceBookingParams struct {
	ID               uuid.UUID     `json:"id"`
	BookingID        uuid.UUID     `json:"booking_id"`
	PartnerServiceID uuid.UUID     `json:"partner_service_id"`
	SlotID           uuid.NullUUID `json:"slot_id"`
	GuestID          uuid.NullUUID `json:"guest_id"`
	Quantity         int16         `json:"quantity"`
	IdempotencyKey   *string       `json:"idempotency_key"`
	PriceCents       *int64        `json:"price_cents"`
}

type UpdateServiceBookingParams struct {
	SlotID     uuid.NullUUID `json:"slot_id"`
	GuestID    uuid.NullUUID `json:"guest_id"`
	Quantity   int16         `json:"quantity"`
	Status     string        `json:"status"`
	PriceCents *int64        `json:"price_cents"`
	ID         uuid.UUID     `json:"id"`
}

type ListServiceBookingsParams struct {
	BookingID uuid.NullUUID `json:"booking_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type GetServiceBookingByIdempotencyParams struct {
	BookingID      uuid.UUID `json:"booking_id"`
	IdempotencyKey *string   `json:"idempotency_key"`
}
