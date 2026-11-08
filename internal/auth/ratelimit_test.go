package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRateLimitsWithDefaultsMatchDocumentedMinimums(t *testing.T) {
	defaults := RateLimits{}.withDefaults()
	if defaults.ExchangeIPPerMin != 10 ||
		defaults.ExchangeIdentityPerMin != 30 ||
		defaults.RefreshIPPerMin != 60 ||
		defaults.RefreshSessionPerMin != 30 ||
		defaults.LinkAccountPerMin != 10 ||
		defaults.LogoutAllAccountPerMin != 5 {
		t.Fatalf("documented rate-limit defaults = %+v", defaults)
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
