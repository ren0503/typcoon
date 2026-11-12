package auth

import (
	"time"

	"github.com/google/uuid"
)

type CreateAccountParams struct {
	ID          uuid.UUID `json:"id"`
	Email       *string   `json:"email"`
	DisplayName *string   `json:"display_name"`
	Phone       *string   `json:"phone"`
	Locale      string    `json:"locale"`
	Status      string    `json:"status"`
}

type LinkAccountIdentityParams struct {
	ID          uuid.UUID `json:"id"`
	AccountID   uuid.UUID `json:"account_id"`
	Provider    string    `json:"provider"`
	ProviderSub string    `json:"provider_sub"`
	Email       *string   `json:"email"`
}

type CreateRefreshTokenParams struct {
	ID          uuid.UUID `json:"id"`
	SubjectType string    `json:"subject_type"`
	SubjectID   uuid.UUID `json:"subject_id"`
	FamilyID    uuid.UUID `json:"family_id"`
	Client      string    `json:"client"`
	TokenHash   []byte    `json:"token_hash"`
	ExpiresAt   time.Time `json:"expires_at"`
	UserAgent   *string   `json:"user_agent"`
}

type Account struct {
	ID          uuid.UUID  `json:"id"`
	Email       *string    `json:"email"`
	DisplayName *string    `json:"display_name"`
	Phone       *string    `json:"phone"`
	Locale      string     `json:"locale"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Version     int64      `json:"version"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

type AccountIdentity struct {
	ID          uuid.UUID `json:"id"`
	AccountID   uuid.UUID `json:"account_id"`
	Provider    string    `json:"provider"`
	ProviderSub string    `json:"provider_sub"`
	Email       *string   `json:"email"`
	CreatedAt   time.Time `json:"created_at"`
}

type AdminUser struct {
	ID          uuid.UUID     `json:"id"`
	TenantID    uuid.UUID     `json:"tenant_id"`
	AccountID   uuid.NullUUID `json:"account_id"`
	Email       string        `json:"email"`
	DisplayName string        `json:"display_name"`
	IdpSubject  *string       `json:"idp_subject"`
	Role        string        `json:"role"`
	MfaEnabled  bool          `json:"mfa_enabled"`
	Status      string        `json:"status"`
	LastLoginAt *time.Time    `json:"last_login_at"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Version     int64         `json:"version"`
	DeletedAt   *time.Time    `json:"deleted_at"`
}

type RefreshToken struct {
	ID          uuid.UUID     `json:"id"`
	SubjectType string        `json:"subject_type"`
	SubjectID   uuid.UUID     `json:"subject_id"`
	FamilyID    uuid.UUID     `json:"family_id"`
	Client      string        `json:"client"`
	TokenHash   []byte        `json:"token_hash"`
	IssuedAt    time.Time     `json:"issued_at"`
	ExpiresAt   time.Time     `json:"expires_at"`
	UsedAt      *time.Time    `json:"used_at"`
	RevokedAt   *time.Time    `json:"revoked_at"`
	ReplacedBy  uuid.NullUUID `json:"replaced_by"`
	UserAgent   *string       `json:"user_agent"`
}
