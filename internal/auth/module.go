package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

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

// Service is the public authentication module interface.
type Service interface {
	Exchange(ctx context.Context, input ExchangeInput) (IssuedSession, error)
	Refresh(ctx context.Context, input RefreshInput) (IssuedSession, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, accountID uuid.UUID) error
	LinkIdentity(ctx context.Context, accountID uuid.UUID, idToken string) (Identity, error)
	UnlinkIdentity(ctx context.Context, accountID, identityID uuid.UUID) error
	JWKS(ctx context.Context) ([]JWK, error)
	Close() error
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

// ServiceOptions supplies all auth orchestration dependencies.
type ServiceOptions struct {
	Store      *Store
	Verifier   Verifier
	Keys       KeySet
	Limiter    RateLimiter
	Clock      Clock
	Audit      audit.Appender
	Logger     *slog.Logger
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
	logger     *slog.Logger
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
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
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
		logger:     logger,
		tokens:     tokens,
		refreshTTL: options.RefreshTTL,
		rateLimits: options.RateLimits.withDefaults(),
	}, nil
}

func (s *service) JWKS(ctx context.Context) (keys []JWK, err error) {
	defer func() { s.logOperation("jwks", "", err) }()
	keys, err = s.keys.PublicJWKS(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	return keys, nil
}

func (s *service) Close() error {
	if s.limiter != nil {
		if err := s.limiter.Close(); err != nil {
			return fmt.Errorf("close rate limiter: %w", err)
		}
	}
	return nil
}

var _ Service = (*service)(nil)
