package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

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

// ServiceOptions supplies all auth orchestration dependencies.
type ServiceOptions struct {
	Store      Store
	Verifier   Verifier
	Keys       KeySet
	Limiter    RateLimiter
	Clock      Clock
	Logger     *slog.Logger
	Issuer     string
	Audience   string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	ClockSkew  time.Duration
	RateLimits RateLimits
}

// NewService builds the auth orchestrator over the production store.
func NewService(options ServiceOptions) (Service, error) {
	if options.Store == nil {
		return nil, errors.New("auth store is required")
	}
	return newService(options, options.Store)
}

func newService(options ServiceOptions, store Store) (Service, error) {
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
		logger:     logger,
		tokens:     tokens,
		refreshTTL: options.RefreshTTL,
		rateLimits: options.RateLimits.withDefaults(),
	}, nil
}

func (s *service) JWKS(ctx context.Context) (keys []JWK, err error) {
	defer func() { s.logOperation(ctx, "jwks", "", err) }()
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
