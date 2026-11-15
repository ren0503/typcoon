package auth

import (
	"context"
	"errors"
	"time"
)

// ErrRateLimiterUnavailable indicates fail-closed limiter behavior.
var ErrRateLimiterUnavailable = errors.New("rate limiter unavailable")

// RateLimitDecision describes the outcome of one rate-limit check.
type RateLimitDecision struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}

// RateLimiter applies a fixed-window counter for an arbitrary key.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (RateLimitDecision, error)
	Close() error
}

// RateLimitedError carries the stable retry indication for a 429 response.
type RateLimitedError struct {
	RetryAfter time.Duration
}

// Error implements error without exposing limiter internals.
func (e *RateLimitedError) Error() string {
	return "authentication rate limited"
}

// Is allows errors.Is(err, ErrRateLimited).
func (e *RateLimitedError) Is(target error) bool {
	return target == ErrRateLimited
}
