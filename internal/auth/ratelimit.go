package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	defaultRedisRateLimitTimeout = 2 * time.Second
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
}

type memoryCounter struct {
	count   int
	resetAt time.Time
}

// MemoryRateLimiter is a process-local fixed-window limiter for development/test.
type MemoryRateLimiter struct {
	mu       sync.Mutex
	clock    Clock
	counters map[string]memoryCounter
}

// NewMemoryRateLimiter builds an in-memory limiter.
func NewMemoryRateLimiter(clock Clock) *MemoryRateLimiter {
	if clock == nil {
		clock = SystemClock{}
	}
	return &MemoryRateLimiter{clock: clock, counters: make(map[string]memoryCounter)}
}

// Allow increments the key's fixed-window counter.
func (l *MemoryRateLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (RateLimitDecision, error) {
	if key == "" || limit <= 0 || window <= 0 {
		return RateLimitDecision{}, errors.New("invalid rate-limit key, limit, or window")
	}
	now := l.clock.Now().UTC()

	l.mu.Lock()
	defer l.mu.Unlock()
	counter, ok := l.counters[key]
	if !ok || !now.Before(counter.resetAt) {
		counter = memoryCounter{resetAt: now.Add(window)}
	}
	counter.count++
	l.counters[key] = counter

	if counter.count > limit {
		return RateLimitDecision{
			Allowed:    false,
			Remaining:  0,
			RetryAfter: counter.resetAt.Sub(now),
		}, nil
	}
	return RateLimitDecision{
		Allowed:   true,
		Remaining: limit - counter.count,
	}, nil
}

// RedisRateLimiter uses Redis atomic fixed-window counters shared by instances.
type RedisRateLimiter struct {
	client  redis.Cmdable
	prefix  string
	timeout time.Duration
}

// NewRedisRateLimiter builds a shared limiter. The prefix namespaces all keys.
func NewRedisRateLimiter(client redis.Cmdable, prefix string) (*RedisRateLimiter, error) {
	return NewRedisRateLimiterWithTimeout(client, prefix, defaultRedisRateLimitTimeout)
}

// NewRedisRateLimiterWithTimeout builds a shared limiter with a bounded Redis call timeout.
func NewRedisRateLimiterWithTimeout(client redis.Cmdable, prefix string, timeout time.Duration) (*RedisRateLimiter, error) {
	if client == nil {
		return nil, errors.New("redis client is required")
	}
	if prefix == "" {
		return nil, errors.New("redis rate-limit prefix is required")
	}
	if timeout <= 0 {
		timeout = defaultRedisRateLimitTimeout
	}
	return &RedisRateLimiter{client: client, prefix: prefix, timeout: timeout}, nil
}

// Allow increments the shared fixed-window counter, failing closed on Redis errors.
func (l *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (RateLimitDecision, error) {
	if key == "" || limit <= 0 || window <= 0 {
		return RateLimitDecision{}, errors.New("invalid rate-limit key, limit, or window")
	}
	callCtx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()
	redisKey := l.prefix + key
	result, err := l.client.Eval(callCtx, redisFixedWindowScript, []string{redisKey}, window.Milliseconds()).Result()
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
	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}
	return RateLimitDecision{Allowed: true, Remaining: remaining}, nil
}

func redisInt(value interface{}) (int64, bool) {
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
	_ RateLimiter = (*MemoryRateLimiter)(nil)
	_ RateLimiter = (*RedisRateLimiter)(nil)
)
