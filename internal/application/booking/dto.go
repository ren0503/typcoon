package booking

import (
	"time"

	"github.com/google/uuid"
	"github.com/ren0503/tycoon/internal/platform/helpers"
)

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

func (input BookingInput) toCreateParams(id uuid.UUID, key *string) CreateBookingParams {
	return CreateBookingParams{
		ID:              id,
		ShipID:          input.ShipID,
		VoyageID:        input.VoyageID,
		AccountID:       helpers.OptionalUUID(input.AccountID),
		RoomID:          helpers.OptionalUUID(input.RoomID),
		EmbarkSeq:       input.EmbarkSeq,
		DisembarkSeq:    input.DisembarkSeq,
		ExpiresAt:       input.ExpiresAt,
		IdempotencyKey:  key,
		SpecialRequests: trimPtr(input.SpecialRequests),
	}
}

// BookingUpdate is the booking update payload. Booking status is immutable.
type BookingUpdate struct {
	RoomID          *uuid.UUID
	EmbarkSeq       *int32
	DisembarkSeq    *int32
	ExpiresAt       *time.Time
	SpecialRequests *string
}

func (input BookingUpdate) toUpdateParam(id uuid.UUID, current Booking) UpdateBookingParams {
	if input.EmbarkSeq != nil {
		current.EmbarkSeq = *input.EmbarkSeq
	}
	if input.DisembarkSeq != nil {
		current.DisembarkSeq = *input.DisembarkSeq
	}
	if input.ExpiresAt != nil {
		current.ExpiresAt = input.ExpiresAt
	}
	if input.SpecialRequests != nil {
		current.SpecialRequests = trimPtr(input.SpecialRequests)
	}

	return UpdateBookingParams{
		ID:              id,
		RoomID:          current.RoomID,
		EmbarkSeq:       current.EmbarkSeq,
		DisembarkSeq:    current.DisembarkSeq,
		ExpiresAt:       current.ExpiresAt,
		SpecialRequests: current.SpecialRequests,
	}
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

func (input ServiceBookingInput) toCreateParam(id uuid.UUID, key *string) CreateServiceBookingParams {
	return CreateServiceBookingParams{
		ID:               id,
		BookingID:        input.BookingID,
		PartnerServiceID: input.PartnerServiceID,
		SlotID:           helpers.OptionalUUID(input.SlotID),
		GuestID:          helpers.OptionalUUID(input.GuestID),
		Quantity:         input.Quantity,
		IdempotencyKey:   key,
		PriceCents:       input.PriceCents,
	}
}

// ServiceBookingUpdate is the service booking update payload.
type ServiceBookingUpdate struct {
	SlotID     *uuid.UUID
	GuestID    *uuid.UUID
	Quantity   *int16
	Status     *string
	PriceCents *int64
}

func (input ServiceBookingUpdate) toUpdateParam(id uuid.UUID, current ServiceBooking) UpdateServiceBookingParams {
	if input.SlotID != nil {
		current.SlotID = helpers.OptionalUUID(input.SlotID)
	}
	if input.GuestID != nil {
		current.GuestID = helpers.OptionalUUID(input.GuestID)
	}
	if input.PriceCents != nil {
		current.PriceCents = input.PriceCents
	}

	return UpdateServiceBookingParams{
		ID:         id,
		SlotID:     current.SlotID,
		GuestID:    current.GuestID,
		Quantity:   current.Quantity,
		Status:     current.Status,
		PriceCents: current.PriceCents,
	}
}
