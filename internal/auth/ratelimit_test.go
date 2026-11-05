package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestMemoryRateLimiterFixedWindow(t *testing.T) {
	clock := &fixedClock{now: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	limiter := NewMemoryRateLimiter(clock)
	ctx := context.Background()

	first, err := limiter.Allow(ctx, "exchange:ip:1.2.3.4", 2, time.Minute)
	if err != nil {
		t.Fatalf("Allow first: %v", err)
	}
	if !first.Allowed || first.Remaining != 1 {
		t.Fatalf("first = %+v", first)
	}
	second, err := limiter.Allow(ctx, "exchange:ip:1.2.3.4", 2, time.Minute)
	if err != nil {
		t.Fatalf("Allow second: %v", err)
	}
	if !second.Allowed || second.Remaining != 0 {
		t.Fatalf("second = %+v", second)
	}
	third, err := limiter.Allow(ctx, "exchange:ip:1.2.3.4", 2, time.Minute)
	if err != nil {
		t.Fatalf("Allow third: %v", err)
	}
	if third.Allowed || third.RetryAfter != time.Minute {
		t.Fatalf("third = %+v", third)
	}

	clock.now = clock.now.Add(time.Minute)
	next, err := limiter.Allow(ctx, "exchange:ip:1.2.3.4", 2, time.Minute)
	if err != nil {
		t.Fatalf("Allow next window: %v", err)
	}
	if !next.Allowed {
		t.Fatalf("next = %+v", next)
	}
}

func TestRedisRateLimiterSharedAndFailsClosed(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	first, err := NewRedisRateLimiter(client, "tycoon:test:")
	if err != nil {
		t.Fatalf("NewRedisRateLimiter: %v", err)
	}
	second, err := NewRedisRateLimiter(client, "tycoon:test:")
	if err != nil {
		t.Fatalf("NewRedisRateLimiter: %v", err)
	}
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		decision, err := first.Allow(ctx, "refresh:1.2.3.4", 2, time.Minute)
		if err != nil {
			t.Fatalf("Allow %d: %v", i, err)
		}
		if !decision.Allowed {
			t.Fatalf("decision %d = %+v", i, decision)
		}
	}
	denied, err := second.Allow(ctx, "refresh:1.2.3.4", 2, time.Minute)
	if err != nil {
		t.Fatalf("Allow denied: %v", err)
	}
	if denied.Allowed || denied.RetryAfter <= 0 {
		t.Fatalf("denied = %+v", denied)
	}

	mr.Close()
	if _, err := first.Allow(ctx, "refresh:1.2.3.4", 2, time.Minute); !errors.Is(err, ErrRateLimiterUnavailable) {
		t.Fatalf("closed Redis error = %v", err)
	}
}
