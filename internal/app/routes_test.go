package app

import (
	"context"
	"io"
	"log/slog"
	"testing"

	httpserver "github.com/ren0503/tycoon/internal/transport/http"
)

type routePinger struct{}

func (routePinger) Ping(context.Context) error { return nil }

func TestBusinessRouteTreeBuilds(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	modules := BuildModules(ModuleDeps{Logger: logger})
	cfg := httpserver.RouterConfig{
		Logger:    logger,
		DB:        routePinger{},
		APIRoutes: modules.RegisterRoutes,
	}
	_ = httpserver.NewRouter(cfg)
}
