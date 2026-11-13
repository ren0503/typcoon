package httpserver

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ren0503/tycoon/internal/transport/http/httpx"
)

type routePinger struct{}

func (routePinger) Ping(context.Context) error { return nil }

func TestRouteInventoryExcludesInternalTables(t *testing.T) {
	cfg := RouterConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:     routePinger{},
		APIRoutes: func(r chi.Router) {
			r.Get("/bookings", func(w http.ResponseWriter, r *http.Request) {
				httpx.WriteJSON(w, http.StatusOK, map[string]string{})
			})
		},
	}
	router := NewRouter(cfg)
	walker, ok := router.(chi.Routes)
	if !ok {
		t.Fatal("router does not implement chi.Routes")
	}
	forbidden := []string{"change-log", "ship-seq", "refresh-tokens", "audit-log", "ops-events", "snapshots", "job-runs"}
	err := chi.Walk(walker, func(_, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		for _, word := range forbidden {
			if strings.Contains(route, word) {
				t.Errorf("forbidden internal route exposed: %s", route)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}
}
