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
