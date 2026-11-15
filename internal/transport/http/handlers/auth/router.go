package auth

import "github.com/go-chi/chi/v5"

// RegisterPublicRoutes mounts auth routes that must bypass bearer authentication.
func RegisterPublicRoutes(r chi.Router, service Service) {
	handler := NewHandler(service)
	r.Post("/auth/firebase/exchange", handler.exchange)
	r.Post("/auth/refresh", handler.refresh)
	r.Post("/auth/logout", handler.logout)
	r.Get("/auth/jwks", handler.jwks)
}

// RegisterProtectedRoutes mounts auth routes behind the global bearer middleware.
func RegisterProtectedRoutes(r chi.Router, service Service) {
	handler := NewHandler(service)
	r.Post("/auth/logout-all", handler.logoutAll)
	r.Post("/auth/identities/link", handler.linkIdentity)
	r.Delete("/auth/identities/{identityId}", handler.unlinkIdentity)
}
