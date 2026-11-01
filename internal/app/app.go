package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ren0503/tycoon/internal/config"
)

// Database is the database lifecycle required by the application.
type Database interface {
	Ping(ctx context.Context) error
	Close()
}

// Server is the HTTP server lifecycle required by the application.
type Server interface {
	Start() error
	Shutdown(ctx context.Context) error
	Addr() string
}

// Monitor is a background health monitor.
type Monitor interface {
	Run(ctx context.Context) error
}

// App orchestrates startup, serving, and graceful shutdown.
type App struct {
	cfg     *config.Config
	logger  *slog.Logger
	db      Database
	server  Server
	monitor Monitor
}

// New creates an application with injected dependencies.
func New(cfg *config.Config, logger *slog.Logger, db Database, server Server, monitor Monitor) *App {
	return &App{
		cfg:     cfg,
		logger:  logger,
		db:      db,
		server:  server,
		monitor: monitor,
	}
}

// Run starts the server and monitor, then blocks until a signal, server failure, or monitor failure.
func (a *App) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- a.server.Start()
	}()

	var monitorErr <-chan error
	if a.monitor != nil {
		ch := make(chan error, 1)
		monitorErr = ch
		go func() {
			ch <- a.monitor.Run(runCtx)
		}()
	}

	a.logger.Info("application started", "addr", a.server.Addr())

	var runErr error
	select {
	case <-ctx.Done():
		a.logger.Info("shutdown signal received")
	case err := <-monitorErr:
		if err != nil {
			runErr = fmt.Errorf("database health monitor failed: %w", err)
			a.logger.Error("database health monitor failed", "error", err)
		}
	case err := <-serverErr:
		if err != nil {
			runErr = fmt.Errorf("http server failed: %w", err)
			a.logger.Error("http server failed", "error", err)
		}
	}

	cancel()

	shutdownErr := a.shutdown()
	if runErr != nil {
		return runErr
	}
	return shutdownErr
}

func (a *App) shutdown() error {
	var errs []error

	if err := a.server.Shutdown(context.Background()); err != nil {
		errs = append(errs, fmt.Errorf("shutdown http server: %w", err))
	}
	a.db.Close()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	a.logger.Info("application stopped")
	return nil
}
