package auth

import (
	"context"

	appauth "github.com/ren0503/tycoon/internal/application/auth"
)

type (
	Service          = appauth.Service
	ExchangeInput    = appauth.ExchangeInput
	RefreshInput     = appauth.RefreshInput
	IssuedSession    = appauth.IssuedSession
	Identity         = appauth.Identity
	JWK              = appauth.JWK
	RateLimitedError = appauth.RateLimitedError
	Principal        = appauth.Principal
	ActorType        = appauth.ActorType
)

const ActorAccount = appauth.ActorAccount

var (
	ErrAuthenticationFailed   = appauth.ErrAuthenticationFailed
	ErrIdentityLinkRequired   = appauth.ErrIdentityLinkRequired
	ErrIdentityConflict       = appauth.ErrIdentityConflict
	ErrLastIdentity           = appauth.ErrLastIdentity
	ErrIdentityNotFound       = appauth.ErrIdentityNotFound
	ErrClientForbidden        = appauth.ErrClientForbidden
	ErrFirebaseUnavailable    = appauth.ErrFirebaseUnavailable
	ErrRateLimiterUnavailable = appauth.ErrRateLimiterUnavailable
	ErrSessionUnavailable     = appauth.ErrSessionUnavailable
)

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	return appauth.PrincipalFrom(ctx)
}
