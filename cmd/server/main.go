package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ren0503/tycoon/internal/app"
	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/infrastructure/aws"
	"github.com/ren0503/tycoon/internal/infrastructure/health"
	"github.com/ren0503/tycoon/internal/infrastructure/observability/logging"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/platform/config"
	"github.com/ren0503/tycoon/internal/transport/http"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		return err
	}

	awsCfg, err := awsconfig.Load(ctx, cfg.AWS)
	if err != nil {
		db.Close()
		return err
	}

	authService, jwtProvider, err := buildAuthService(ctx, cfg, db, logger, awsCfg)
	if err != nil {
		db.Close()
		return err
	}
	defer func() { _ = authService.Close() }()

	provider, err := authProvider(cfg, jwtProvider)
	if err != nil {
		db.Close()
		return err
	}

	modules := app.BuildModules(db, logger, authService)

	router := httpserver.NewRouter(httpserver.RouterConfig{
		Logger:             logger,
		DB:                 db,
		CORSAllowedOrigins: cfg.CORS.AllowedOrigins,
		Provider:           provider,
		PublicAPIRoutes:    modules.RegisterPublicRoutes,
		APIRoutes:          modules.RegisterRoutes,
	})
	server := httpserver.NewServer(cfg, router)
	monitor := health.New(db, logger)

	application := app.New(cfg, logger, db, server, monitor)
	return application.Run(ctx)
}

func authProvider(cfg *config.Config, jwtProvider auth.Provider) (auth.Provider, error) {
	if cfg.Auth.Mode == "dev" {
		return auth.NewDevHeaderProvider(cfg.Auth.DevPrincipals)
	}
	if jwtProvider != nil {
		return jwtProvider, nil
	}
	return auth.DenyAllProvider{}, nil
}
