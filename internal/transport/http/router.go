package httpserver

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/ren0503/tycoon/internal/transport/http/handlers/health"
	custommiddleware "github.com/ren0503/tycoon/internal/transport/http/middleware"
)

// RouterConfig holds the dependencies required to build the HTTP router.
type RouterConfig struct {
	Logger             *slog.Logger
	DB                 health.Pinger
	CORSAllowedOrigins []string
	Provider           custommiddleware.Provider
	PublicAPIRoutes    func(r chi.Router)
	APIRoutes          func(r chi.Router)
}

// NewRouter builds the Chi router with technical routes and foundation middleware.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(custommiddleware.RequestLogger(cfg.Logger))
	r.Use(custommiddleware.Recoverer(cfg.Logger))
	r.Use(custommiddleware.CORS(cfg.CORSAllowedOrigins))

	health.Register(r, cfg.DB, cfg.Logger)

	if cfg.PublicAPIRoutes != nil || cfg.APIRoutes != nil {
		r.Route("/api", func(r chi.Router) {
			if cfg.PublicAPIRoutes != nil {
				cfg.PublicAPIRoutes(r)
			}
			if cfg.APIRoutes != nil {
				provider := cfg.Provider
				if provider == nil {
					provider = custommiddleware.DenyAllProvider{}
				}
				r.Group(func(r chi.Router) {
					r.Use(custommiddleware.Authenticate(provider, cfg.Logger))
					cfg.APIRoutes(r)
				})
			}
		})
	}

	return r
}
