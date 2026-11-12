package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

const readinessTimeout = 2 * time.Second

// Pinger reports database connectivity.
type Pinger interface {
	Ping(ctx context.Context) error
}

type statusResponse struct {
	Status string `json:"status"`
}

// Health is a dependency-free liveness handler.
func Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}

// Ready reports whether the instance can serve traffic by checking the postgres.
func Ready(db Pinger, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			logger.Warn("readiness check failed",
				"request_id", middleware.GetReqID(r.Context()),
				"error", err,
			)
			writeJSON(w, http.StatusServiceUnavailable, statusResponse{Status: "not_ready"})
			return
		}

		writeJSON(w, http.StatusOK, statusResponse{Status: "ready"})
	}
}
