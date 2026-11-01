package health

import (
	"context"
	"log/slog"
	"time"
)

// Pinger reports database connectivity.
type Pinger interface {
	Ping(ctx context.Context) error
}

const (
	// DefaultInterval is how often the database is checked.
	DefaultInterval = 5 * time.Second
	// DefaultTimeout bounds a single database check.
	DefaultTimeout = 3 * time.Second
)

// Checker periodically verifies database connectivity and returns an error when it is lost.
type Checker struct {
	db       Pinger
	interval time.Duration
	timeout  time.Duration
	logger   *slog.Logger
}

// New creates a database health checker.
func New(db Pinger, logger *slog.Logger) *Checker {
	return &Checker{
		db:       db,
		interval: DefaultInterval,
		timeout:  DefaultTimeout,
		logger:   logger,
	}
}

// Run checks the database until the context is canceled or connectivity is lost.
func (c *Checker) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, c.timeout)
			err := c.db.Ping(pingCtx)
			cancel()
			if err != nil {
				c.logger.Error("database health check failed", "error", err)
				return err
			}
		}
	}
}
