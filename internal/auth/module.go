package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/audit"
)

// ErrAuthNotImplemented marks foundational service stubs.
var ErrAuthNotImplemented = errors.New("auth service not implemented")

// ExchangeInput is one Firebase-to-backend-session request.
type ExchangeInput struct {
	IDToken   string
	Client    string
	UserAgent string
	IPAddress string
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

// Service is the public authentication module interface.
type Service interface {
	Exchange(ctx context.Context, input ExchangeInput) (IssuedSession, error)
	Refresh(ctx context.Context, input RefreshInput) (IssuedSession, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, accountID uuid.UUID) error
	LinkIdentity(ctx context.Context, accountID uuid.UUID, idToken string) (Identity, error)
	UnlinkIdentity(ctx context.Context, accountID, identityID uuid.UUID) error
	JWKS(ctx context.Context) ([]JWK, error)
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
	if r.ExchangeIPPerMin <= 0 {
		r.ExchangeIPPerMin = defaultExchangeIPPerMin
	}
	if r.ExchangeIdentityPerMin <= 0 {
		r.ExchangeIdentityPerMin = defaultExchangeIdentityPerMin
	}
	if r.RefreshIPPerMin <= 0 {
		r.RefreshIPPerMin = defaultRefreshIPPerMin
	}
	if r.RefreshSessionPerMin <= 0 {
		r.RefreshSessionPerMin = defaultRefreshSessionPerMin
	}
	if r.LinkAccountPerMin <= 0 {
		r.LinkAccountPerMin = defaultLinkAccountPerMin
	}
	if r.LogoutAllAccountPerMin <= 0 {
		r.LogoutAllAccountPerMin = defaultLogoutAllAccountPerMin
	}
	return r
}

// ServiceOptions supplies all auth orchestration dependencies.
type ServiceOptions struct {
	Store      *Store
	Verifier   Verifier
	Keys       KeySet
	Limiter    RateLimiter
	Clock      Clock
	Audit      audit.Appender
	Issuer     string
	Audience   string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	ClockSkew  time.Duration
	RateLimits RateLimits
}

type service struct {
	store      serviceStore
	verifier   Verifier
	keys       KeySet
	limiter    RateLimiter
	clock      Clock
	audit      audit.Appender
	tokens     *AccessTokenIssuer
	refreshTTL time.Duration
	rateLimits RateLimits
}

// NewService builds the auth orchestrator over the production store.
func NewService(options ServiceOptions) (Service, error) {
	if options.Store == nil {
		return nil, errors.New("auth store is required")
	}
	return newService(options, options.Store)
}

func newService(options ServiceOptions, store serviceStore) (Service, error) {
	if store == nil {
		return nil, errors.New("auth store is required")
	}
	if options.Verifier == nil {
		return nil, errors.New("firebase verifier is required")
	}
	if options.Keys == nil {
		return nil, errors.New("signing key set is required")
	}
	if options.Limiter == nil {
		return nil, errors.New("rate limiter is required")
	}
	if options.Audit == nil {
		return nil, errors.New("audit appender is required")
	}
	if options.RefreshTTL <= 0 {
		return nil, errors.New("refresh token TTL must be positive")
	}
	clock := options.Clock
	if clock == nil {
		clock = SystemClock{}
	}
	tokens, err := NewAccessTokenIssuer(
		options.Keys,
		options.Issuer,
		options.Audience,
		options.AccessTTL,
		options.ClockSkew,
		clock,
	)
	if err != nil {
		return nil, err
	}
	return &service{
		store:      store,
		verifier:   options.Verifier,
		keys:       options.Keys,
		limiter:    options.Limiter,
		clock:      clock,
		audit:      options.Audit,
		tokens:     tokens,
		refreshTTL: options.RefreshTTL,
		rateLimits: options.RateLimits.withDefaults(),
	}, nil
}

func (s *service) JWKS(context.Context) ([]JWK, error) {
	return nil, ErrAuthNotImplemented
}

var _ Service = (*service)(nil)
