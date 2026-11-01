package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Register installs the technical routes and the foundation error handlers.
func Register(r chi.Router, db Pinger, logger *slog.Logger) {
	r.Get("/health", Health)
	r.Get("/ready", Ready(db, logger))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotFound, "not_found", "Resource not found.")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
	})
}
