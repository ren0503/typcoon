//go:build integration

package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
	"github.com/ren0503/tycoon/internal/testutil"
)

func TestGeneratedPingQuery(t *testing.T) {
	cfg := testutil.PostgresConfig(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.New(ctx, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	value, err := sqlc.New(pool).Ping(ctx)
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if value != 1 {
		t.Fatalf("Ping() = %d", value)
	}
}
