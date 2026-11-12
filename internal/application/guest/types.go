package guest

import (
	"time"

	"github.com/google/uuid"
)

type Guest struct {
	ID                  uuid.UUID  `json:"id"`
	BookingID           uuid.UUID  `json:"booking_id"`
	FullName            string     `json:"full_name"`
	DateOfBirth         *time.Time `json:"date_of_birth"`
	Nationality         *string    `json:"nationality"`
	DocumentType        *string    `json:"document_type"`
	DocumentEnc         []byte     `json:"document_enc"`
	DocumentKeyID       *string    `json:"document_key_id"`
	EmergencyContactEnc []byte     `json:"emergency_contact_enc"`
	SpecialNeeds        *string    `json:"special_needs"`
	IsLead              bool       `json:"is_lead"`
	MusterStation       *string    `json:"muster_station"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	Version             int64      `json:"version"`
	DeletedAt           *time.Time `json:"deleted_at"`
}

type BoardingPass struct {
	ID        uuid.UUID     `json:"id"`
	BookingID uuid.UUID     `json:"booking_id"`
	GuestID   uuid.NullUUID `json:"guest_id"`
	Jti       uuid.UUID     `json:"jti"`
	IssuedAt  time.Time     `json:"issued_at"`
	ExpiresAt time.Time     `json:"expires_at"`
	RevokedAt *time.Time    `json:"revoked_at"`
}

type CreateGuestParams struct {
	ID                  uuid.UUID  `json:"id"`
	BookingID           uuid.UUID  `json:"booking_id"`
	FullName            string     `json:"full_name"`
	DateOfBirth         *time.Time `json:"date_of_birth"`
	Nationality         *string    `json:"nationality"`
	DocumentType        *string    `json:"document_type"`
	DocumentEnc         []byte     `json:"document_enc"`
	DocumentKeyID       *string    `json:"document_key_id"`
	EmergencyContactEnc []byte     `json:"emergency_contact_enc"`
	SpecialNeeds        *string    `json:"special_needs"`
	IsLead              bool       `json:"is_lead"`
	MusterStation       *string    `json:"muster_station"`
}

type UpdateGuestParams struct {
	FullName            string     `json:"full_name"`
	DateOfBirth         *time.Time `json:"date_of_birth"`
	Nationality         *string    `json:"nationality"`
	DocumentType        *string    `json:"document_type"`
	DocumentEnc         []byte     `json:"document_enc"`
	DocumentKeyID       *string    `json:"document_key_id"`
	EmergencyContactEnc []byte     `json:"emergency_contact_enc"`
	SpecialNeeds        *string    `json:"special_needs"`
	IsLead              bool       `json:"is_lead"`
	MusterStation       *string    `json:"muster_station"`
	ID                  uuid.UUID  `json:"id"`
}

type ListGuestsParams struct {
	BookingID uuid.UUID     `json:"booking_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateBoardingPassParams struct {
	ID        uuid.UUID     `json:"id"`
	BookingID uuid.UUID     `json:"booking_id"`
	GuestID   uuid.NullUUID `json:"guest_id"`
	Jti       uuid.UUID     `json:"jti"`
	ExpiresAt time.Time     `json:"expires_at"`
}

type ListBoardingPassesParams struct {
	BookingID uuid.UUID     `json:"booking_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}
