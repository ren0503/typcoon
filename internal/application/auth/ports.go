package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/audit"
)

// AdminAuthority is the backend-resolved Fleet Console authorization context.
type AdminAuthority struct {
	AdminUserID uuid.UUID
	Role        string
	TenantID    uuid.UUID
	ShipIDs     []uuid.UUID
}

// VerifiedIdentity is the verified subset of a Firebase ID token used by auth.
type VerifiedIdentity struct {
	UID           string
	Email         string
	EmailVerified bool
	DisplayName   string
}

// Verifier verifies a Firebase ID Token and returns stable identity fields.
type Verifier interface {
	Verify(ctx context.Context, idToken string) (VerifiedIdentity, error)
}

// AccountLookup resolves active accounts and administrator authority.
type AccountLookup interface {
	GetAccount(ctx context.Context, id uuid.UUID) (Account, error)
	ResolveActiveAdmin(ctx context.Context, accountID uuid.UUID) (AdminAuthority, error)
}

// Store owns transactional persistence for authentication state.
type Store interface {
	AppendAudit(context.Context, audit.Entry) error
	GetIdentityByProviderSubject(ctx context.Context, provider, subject string) (AccountIdentity, error)
	GetAccount(ctx context.Context, id uuid.UUID) (Account, error)
	ProvisionGuest(ctx context.Context, account CreateAccountParams, identity LinkAccountIdentityParams, entry audit.Entry) (Account, AccountIdentity, error)
	CreateSession(ctx context.Context, token CreateRefreshTokenParams, entry audit.Entry) error
	GetRefreshTokenByHash(ctx context.Context, hash []byte) (RefreshToken, error)
	ResolveActiveAdmin(ctx context.Context, accountID uuid.UUID) (AdminAuthority, error)
	RevokeRefreshFamilyByHash(ctx context.Context, hash []byte, entry audit.Entry) error
	RevokeAccountSessions(ctx context.Context, accountID uuid.UUID, entry audit.Entry) error
	LinkIdentityForAccount(ctx context.Context, accountID uuid.UUID, identity LinkAccountIdentityParams, entry audit.Entry) (AccountIdentity, error)
	UnlinkIdentityForAccount(ctx context.Context, accountID, identityID uuid.UUID, entry audit.Entry) error
	RotateRefreshToken(ctx context.Context, hash []byte, replacement CreateRefreshTokenParams, refreshedEntry, reuseEntry audit.Entry, now time.Time) (Account, RefreshToken, error)
}

var (
	ErrIdentityNotFound     = errors.New("account identity not found")
	ErrIdentityConflict     = errors.New("account identity already exists")
	ErrEmailConflict        = errors.New("account email already exists")
	ErrAdminGrantNotFound   = errors.New("active administrator grant not found")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenInvalid  = errors.New("refresh token invalid")
	ErrRefreshTokenReused   = errors.New("refresh token reuse detected")
	ErrInvalidFirebaseToken = errors.New("invalid Firebase ID token")
	ErrFirebaseUnavailable  = errors.New("firebase verification unavailable")
)
