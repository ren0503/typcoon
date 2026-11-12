package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/transport/http/httpx"
)

// Authenticate resolves the request principal and stores it in the context.
// Requests without a resolvable principal are rejected with 401.
func Authenticate(provider auth.Provider, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := provider.Authenticate(r)
			if err != nil {
				if errors.Is(err, auth.ErrUnauthenticated) {
					logger.Info("auth outcome", "operation", "authenticate", "outcome", "unauthenticated")
				} else {
					logger.Warn("auth outcome", "operation", "authenticate", "outcome", "dependency_unavailable")
				}
				httpx.WriteError(w, r, apperrors.Unauthorized("Authentication is required."))
				return
			}
			if _, ok := provider.(*auth.DevHeaderProvider); ok {
				logger.Info("dev-principal accepted", "actor_type", principal.ActorType, "actor_id", principal.ActorID)
			}
			next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
		})
	}
}
