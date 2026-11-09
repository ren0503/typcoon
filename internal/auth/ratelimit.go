package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrRateLimiterUnavailable indicates fail-closed limiter behavior.
var ErrRateLimiterUnavailable = errors.New("rate limiter unavailable")

const (
	redisFixedWindowScript = `
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('PTTL', KEYS[1])
return {current, ttl}
`
)

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

// RedisRateLimiter uses Redis atomic fixed-window counters shared by instances.
type RedisRateLimiter struct {
	client *redis.Client
	prefix string
}

// NewRedisRateLimiter builds a shared limiter. The prefix namespaces all keys.
func NewRedisRateLimiter(client *redis.Client, prefix string) (*RedisRateLimiter, error) {
	if client == nil {
		return nil, errors.New("redis client is required")
	}
	if prefix == "" {
		return nil, errors.New("redis rate-limit prefix is required")
	}

	return &RedisRateLimiter{client: client, prefix: prefix}, nil
}

// Allow increments the shared fixed-window counter, failing closed on Redis errors.
func (l *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (RateLimitDecision, error) {
	if key == "" || limit <= 0 || window <= 0 {
		return RateLimitDecision{}, errors.New("invalid rate-limit key, limit, or window")
	}
	redisKey := l.prefix + key
	result, err := l.client.Eval(ctx, redisFixedWindowScript, []string{redisKey}, window.Milliseconds()).Result()
	if err != nil {
		return RateLimitDecision{}, fmt.Errorf("%w: %v", ErrRateLimiterUnavailable, err)
	}
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return RateLimitDecision{}, fmt.Errorf("%w: unexpected Redis result", ErrRateLimiterUnavailable)
	}
	count, ok := redisInt(values[0])
	if !ok {
		return RateLimitDecision{}, fmt.Errorf("%w: invalid Redis count", ErrRateLimiterUnavailable)
	}
	ttlMS, ok := redisInt(values[1])
	if !ok {
		return RateLimitDecision{}, fmt.Errorf("%w: invalid Redis TTL", ErrRateLimiterUnavailable)
	}
	if count > int64(limit) {
		retryAfter := time.Duration(ttlMS) * time.Millisecond
		if retryAfter <= 0 {
			retryAfter = window
		}
		return RateLimitDecision{Allowed: false, Remaining: 0, RetryAfter: retryAfter}, nil
	}
	remaining := max(limit - int(count), 0)
	return RateLimitDecision{Allowed: true, Remaining: remaining}, nil
}

func (l *RedisRateLimiter) Close() error {
	return l.client.Close()
}

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

var (
	_ RateLimiter = (*RedisRateLimiter)(nil)
)
