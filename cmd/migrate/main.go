package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ren0503/tycoon/internal/infrastructure/observability/logging"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/platform/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel, cfg.AppEnv)
	slog.SetDefault(logger)

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := postgres.RunMigrations(ctx, postgres.DSN(cfg.Database), command); err != nil {
		return err
	}

	logger.Info("migration command completed", "command", command)
	return nil
}
