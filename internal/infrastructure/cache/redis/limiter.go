// Package redis implements shared authentication rate limiting.
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	appauth "github.com/ren0503/tycoon/internal/application/auth"
)

const redisFixedWindowScript = `
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('PTTL', KEYS[1])
return {current, ttl}
`

// RateLimiter uses Redis atomic fixed-window counters shared by instances.
type RateLimiter struct {
	client *goredis.Client
	prefix string
}

// NewRateLimiter builds a shared limiter. The prefix namespaces all keys.
func NewRateLimiter(client *goredis.Client, prefix string) (*RateLimiter, error) {
	if client == nil {
		return nil, errors.New("redis client is required")
	}
	if prefix == "" {
		return nil, errors.New("redis rate-limit prefix is required")
	}
	return &RateLimiter{client: client, prefix: prefix}, nil
}

// Allow increments the shared fixed-window counter, failing closed on Redis errors.
func (l *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (appauth.RateLimitDecision, error) {
	if key == "" || limit <= 0 || window <= 0 {
		return appauth.RateLimitDecision{}, errors.New("invalid rate-limit key, limit, or window")
	}
	result, err := l.client.Eval(ctx, redisFixedWindowScript, []string{l.prefix + key}, window.Milliseconds()).Result()
	if err != nil {
		return appauth.RateLimitDecision{}, fmt.Errorf("%w: %v", appauth.ErrRateLimiterUnavailable, err)
	}
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return appauth.RateLimitDecision{}, fmt.Errorf("%w: unexpected Redis result", appauth.ErrRateLimiterUnavailable)
	}
	count, ok := redisInt(values[0])
	if !ok {
		return appauth.RateLimitDecision{}, fmt.Errorf("%w: invalid Redis count", appauth.ErrRateLimiterUnavailable)
	}
	ttlMS, ok := redisInt(values[1])
	if !ok {
		return appauth.RateLimitDecision{}, fmt.Errorf("%w: invalid Redis TTL", appauth.ErrRateLimiterUnavailable)
	}
	if count > int64(limit) {
		retryAfter := time.Duration(ttlMS) * time.Millisecond
		if retryAfter <= 0 {
			retryAfter = window
		}
		return appauth.RateLimitDecision{Allowed: false, Remaining: 0, RetryAfter: retryAfter}, nil
	}
	remaining := max(limit-int(count), 0)
	return appauth.RateLimitDecision{Allowed: true, Remaining: remaining}, nil
}

// Close closes the underlying Redis client.
func (l *RateLimiter) Close() error { return l.client.Close() }

func redisInt(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case string:
		var parsed int64
		_, err := fmt.Sscan(typed, &parsed)
		return parsed, err == nil
	default:
		return 0, false
	}
}

var _ appauth.RateLimiter = (*RateLimiter)(nil)
