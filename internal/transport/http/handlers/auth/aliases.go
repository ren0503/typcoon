// Package auth exposes the authentication HTTP handlers.
package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	appauth "github.com/ren0503/tycoon/internal/application/auth"
)

// Backend auth types re-exported for the handlers in this package.
type (
	// Service is the authentication use-case surface.
	Service = appauth.Service
	// ExchangeInput is a Firebase-to-backend-session request.
	ExchangeInput = appauth.ExchangeInput
	// RefreshInput is a refresh-token request.
	RefreshInput = appauth.RefreshInput
	// IssuedSession is the result of a successful exchange or refresh.
	IssuedSession = appauth.IssuedSession
	// Identity is a linked external identity.
	Identity = appauth.Identity
	// JWK is a public signing key.
	JWK = appauth.JWK
	// RateLimitedError reports a rate-limited auth operation.
	RateLimitedError = appauth.RateLimitedError
	// Principal is the authenticated caller.
	Principal = appauth.Principal
	// ActorType identifies who performed an action.
	ActorType = appauth.ActorType
)

// ActorAccount identifies an account actor.
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

func AccountIDFromPrincipal(r *http.Request) (uuid.UUID, bool) {
	principal, ok := PrincipalFrom(r.Context())
	if !ok {
		return uuid.Nil, false
	}
	accountID := principal.AccountID
	if accountID == uuid.Nil && principal.ActorType == ActorAccount {
		accountID, _ = uuid.Parse(principal.ActorID)
	}
	return accountID, accountID != uuid.Nil
}
