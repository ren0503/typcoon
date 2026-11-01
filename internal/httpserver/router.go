package httpserver

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/ren0503/tycoon/internal/httpserver/handlers"
	custommiddleware "github.com/ren0503/tycoon/internal/httpserver/middleware"
)

// RouterConfig holds the dependencies required to build the HTTP router.
type RouterConfig struct {
	Logger             *slog.Logger
	DB                 handlers.Pinger
	CORSAllowedOrigins []string
}

// NewRouter builds the Chi router with technical routes and foundation middleware.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(custommiddleware.RequestLogger(cfg.Logger))
	r.Use(custommiddleware.Recoverer(cfg.Logger))
	r.Use(custommiddleware.CORS(cfg.CORSAllowedOrigins))

	handlers.Register(r, cfg.DB, cfg.Logger)

	return r
}
