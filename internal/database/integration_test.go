//go:build integration

package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/testutil"
)

func TestNewPingsDatabase(t *testing.T) {
	cfg := testutil.PostgresConfig(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.New(ctx, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestRunMigrations(t *testing.T) {
	dsn := testutil.PostgresDSN(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.RunMigrations(ctx, dsn, "up"); err != nil {
		t.Fatalf("up migration error = %v", err)
	}
	if err := database.RunMigrations(ctx, dsn, "status"); err != nil {
		t.Fatalf("status migration error = %v", err)
	}
	if err := database.RunMigrations(ctx, dsn, "down"); err != nil {
		t.Fatalf("down migration error = %v", err)
	}
}
