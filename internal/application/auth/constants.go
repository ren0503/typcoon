package auth

import (
	"errors"
	"time"
)

const (
	firebaseProvider = "firebase"
	defaultLocale    = "en"
	rateLimitWindow  = time.Minute
)

var (
	// ErrAuthenticationFailed deliberately hides why an exchange credential failed.
	ErrAuthenticationFailed = errors.New("authentication failed")
	// ErrIdentityLinkRequired means a verified email already belongs to another account.
	ErrIdentityLinkRequired = errors.New("identity link required")
	// ErrClientForbidden means the requested session class is not authorized.
	ErrClientForbidden = errors.New("auth client forbidden")
	// ErrRateLimited indicates an authentication abuse limit was exceeded.
	ErrRateLimited = errors.New("auth rate limited")
	// ErrSessionUnavailable indicates signing or key material was unavailable.
	ErrSessionUnavailable = errors.New("auth session issuer unavailable")
	// ErrLastIdentity means unlinking would leave the account without a sign-in method.
	ErrLastIdentity = errors.New("last sign-in identity")
)
