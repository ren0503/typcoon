package app

import (
	"context"
	"log/slog"

	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/platform/config"
)

// RunMigrations assembles and runs the migration runtime. It initializes only
// PostgreSQL configuration and Goose; no auth, HTTP, Firebase, Redis, or KMS
// dependency is constructed.
func RunMigrations(ctx context.Context, cfg *config.Config, logger *slog.Logger, command string) error {
	if err := postgres.RunMigrations(ctx, postgres.DSN(cfg.Database), command); err != nil {
		return err
	}
	logger.Info("migration command completed", "command", command)
	return nil
}
