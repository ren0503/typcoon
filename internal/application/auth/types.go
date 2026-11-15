package auth

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

// CreateAccountParams is the input for creating an account.
type CreateAccountParams struct {
	ID          uuid.UUID `json:"id"`
	Email       *string   `json:"email"`
	DisplayName *string   `json:"display_name"`
	Phone       *string   `json:"phone"`
	Locale      string    `json:"locale"`
	Status      string    `json:"status"`
}

// LinkAccountIdentityParams is the input for linking an identity to an account.
type LinkAccountIdentityParams struct {
	ID          uuid.UUID `json:"id"`
	AccountID   uuid.UUID `json:"account_id"`
	Provider    string    `json:"provider"`
	ProviderSub string    `json:"provider_sub"`
	Email       *string   `json:"email"`
}

// CreateRefreshTokenParams is the input for persisting a refresh token hash.
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

// Account is a backend-owned account record.
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

// AccountIdentity links an external identity to an account.
type AccountIdentity struct {
	ID          uuid.UUID `json:"id"`
	AccountID   uuid.UUID `json:"account_id"`
	Provider    string    `json:"provider"`
	ProviderSub string    `json:"provider_sub"`
	Email       *string   `json:"email"`
	CreatedAt   time.Time `json:"created_at"`
}

// AdminUser is a member of the administrative console.
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

// RefreshToken is a rotated refresh-token record.
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

// ExchangeInput is one Firebase-to-backend-session request.
type ExchangeInput struct {
	IDToken   string
	Client    string
	UserAgent string
	IPAddress string
}

// Valid reports whether the exchange input is well formed.
func (e ExchangeInput) Valid() error {
	if e.Client != "" && (e.Client != ClientGuestMobile && e.Client != ClientFleetConsole) {
		return ErrClientForbidden
	}

	if strings.TrimSpace(e.IDToken) == "" {
		return ErrAuthenticationFailed
	}

	return nil
}

// RefreshInput is one refresh-token rotation request.
type RefreshInput struct {
	RefreshToken string
	UserAgent    string
	IPAddress    string
}

// IssuedSession is the backend session and token pair returned to a client.
type IssuedSession struct {
	SessionID        uuid.UUID
	AccountID        uuid.UUID
	Client           string
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// Identity is a linked provider identity exposed by auth.
type Identity struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	Provider  string
	Email     *string
	CreatedAt time.Time
}

// RateLimits holds the documented per-minute authentication limits.
type RateLimits struct {
	ExchangeIPPerMin       int
	ExchangeIdentityPerMin int
	RefreshIPPerMin        int
	RefreshSessionPerMin   int
	LinkAccountPerMin      int
	LogoutAllAccountPerMin int
}

const (
	defaultExchangeIPPerMin       = 10
	defaultExchangeIdentityPerMin = 30
	defaultRefreshIPPerMin        = 60
	defaultRefreshSessionPerMin   = 30
	defaultLinkAccountPerMin      = 10
	defaultLogoutAllAccountPerMin = 5
)

func (r RateLimits) withDefaults() RateLimits {
	r.ExchangeIPPerMin = lo.CoalesceOrEmpty(r.ExchangeIPPerMin, defaultExchangeIPPerMin)
	r.ExchangeIdentityPerMin = lo.CoalesceOrEmpty(r.ExchangeIdentityPerMin, defaultExchangeIdentityPerMin)
	r.RefreshIPPerMin = lo.CoalesceOrEmpty(r.RefreshIPPerMin, defaultRefreshIPPerMin)
	r.RefreshSessionPerMin = lo.CoalesceOrEmpty(r.RefreshSessionPerMin, defaultRefreshSessionPerMin)
	r.LinkAccountPerMin = lo.CoalesceOrEmpty(r.LinkAccountPerMin, defaultLinkAccountPerMin)
	r.LogoutAllAccountPerMin = lo.CoalesceOrEmpty(r.LogoutAllAccountPerMin, defaultLogoutAllAccountPerMin)
	return r
}
