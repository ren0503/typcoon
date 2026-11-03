package guest

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/auth"
)

// DocumentCipher seals and validates guest document ciphertext. Implementations
// must never expose plaintext; production uses an envelope/KMS implementation
// while LocalDocumentCipher provides a non-KMS implementation for dev and tests.
type DocumentCipher interface {
	Encrypt(plaintext []byte) (ciphertext []byte, keyID string, err error)
	Validate(ciphertext []byte) error
}

const documentEnvelopeMagic = "TYC1"

// LocalDocumentCipher is a non-KMS AES-GCM document cipher for development and
// tests. Ciphertext carries a versioned envelope so plaintext can be rejected.
type LocalDocumentCipher struct {
	keyID string
	aead  cipher.AEAD
}

// NewLocalDocumentCipher builds a local cipher with the supplied 16/24/32-byte key.
func NewLocalDocumentCipher(keyID string, key []byte) (*LocalDocumentCipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if keyID == "" {
		keyID = "local-dev"
	}
	return &LocalDocumentCipher{keyID: keyID, aead: aead}, nil
}

// NewDevDocumentCipher builds a local cipher with a random ephemeral key.
func NewDevDocumentCipher() (*LocalDocumentCipher, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return NewLocalDocumentCipher("local-dev", key)
}

// Encrypt seals plaintext into the versioned local envelope.
func (c *LocalDocumentCipher) Encrypt(plaintext []byte) ([]byte, string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, "", err
	}
	sealed := c.aead.Seal(nil, nonce, plaintext, []byte(documentEnvelopeMagic))
	out := make([]byte, 0, len(documentEnvelopeMagic)+len(nonce)+len(sealed))
	out = append(out, documentEnvelopeMagic...)
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, c.keyID, nil
}

// Validate rejects anything that is not a structurally valid ciphertext envelope.
func (c *LocalDocumentCipher) Validate(ciphertext []byte) error {
	if len(ciphertext) < len(documentEnvelopeMagic)+c.aead.NonceSize()+c.aead.Overhead() {
		return errors.New("ciphertext is too short")
	}
	if string(ciphertext[:len(documentEnvelopeMagic)]) != documentEnvelopeMagic {
		return errors.New("ciphertext envelope is not recognised")
	}
	return nil
}

func validateCiphertext(c DocumentCipher, value []byte, field string) error {
	if len(value) == 0 {
		return nil
	}
	if c == nil {
		return apperrors.Internal(errors.New("document cipher is not configured"))
	}
	if err := c.Validate(value); err != nil {
		return apperrors.Validation(apperrors.CodeValidationFailed, field+" must be opaque ciphertext produced by the encryption interface.")
	}
	return nil
}

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
