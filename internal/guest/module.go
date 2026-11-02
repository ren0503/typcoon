package guest

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ren0503/tycoon/internal/auth"
)

// Page is a cursor-paginated collection.
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// Input is the guest creation payload. Document fields are opaque ciphertext.
type Input struct {
	FullName            string
	DateOfBirth         *time.Time
	Nationality         *string
	DocumentType        *string
	DocumentEnc         []byte
	DocumentKeyID       *string
	EmergencyContactEnc []byte
	SpecialNeeds        *string
	IsLead              bool
	MusterStation       *string
}

// Update is the guest update payload.
type Update struct {
	FullName      *string
	SpecialNeeds  *string
	IsLead        *bool
	MusterStation *string
	DocumentType  *string
}

// View is the sanitized guest representation.
type View struct {
	ID                  uuid.UUID  `json:"id"`
	BookingID           uuid.UUID  `json:"booking_id"`
	FullName            string     `json:"full_name"`
	DateOfBirth         *string    `json:"date_of_birth,omitempty"`
	Nationality         *string    `json:"nationality"`
	DocumentType        *string    `json:"document_type"`
	DocumentKeyID       *string    `json:"document_key_id"`
	HasDocument         bool       `json:"has_document"`
	HasEmergencyContact bool       `json:"has_emergency_contact"`
	SpecialNeeds        *string    `json:"special_needs"`
	IsLead              bool       `json:"is_lead"`
	MusterStation       *string    `json:"muster_station"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	Version             int64      `json:"version"`
	DeletedAt           *time.Time `json:"deleted_at"`
}

// Service is the public surface of the guest module.
type Service interface {
	Create(ctx context.Context, principal auth.Principal, bookingID uuid.UUID, input Input) (View, error)
	Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (View, error)
	List(ctx context.Context, principal auth.Principal, bookingID uuid.UUID, cursor string, limit int32) (Page[View], error)
	Update(ctx context.Context, principal auth.Principal, id uuid.UUID, input Update) (View, error)
	Delete(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	IssueBoardingPass(ctx context.Context, principal auth.Principal, bookingID uuid.UUID, guestID *uuid.UUID, expiresAt *time.Time) (BoardingPassView, error)
	GetBoardingPass(ctx context.Context, principal auth.Principal, id uuid.UUID) (BoardingPassView, error)
	ListBoardingPasses(ctx context.Context, principal auth.Principal, bookingID uuid.UUID, cursor string, limit int32) (Page[BoardingPassView], error)
	RevokeBoardingPass(ctx context.Context, principal auth.Principal, id uuid.UUID) (BoardingPassView, error)
}

func dateValue(date pgtype.Date) *string {
	if !date.Valid {
		return nil
	}
	text := date.Time.Format("2006-01-02")
	return &text
}

func decodeCiphertext(raw *string) ([]byte, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(*raw)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}
