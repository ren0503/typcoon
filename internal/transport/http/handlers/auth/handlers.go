package auth

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/transport/http/httpx"
)

// Handler owns the public and protected auth HTTP surface.
type Handler struct {
	service Service
}

// NewHandler builds auth handlers over the public service.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

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

func (h *Handler) exchange(w http.ResponseWriter, r *http.Request) {
	httpx.SetNoStore(w)
	var request FirebaseExchangeRequest
	if err := httpx.Decode(w, r, &request); err != nil {
		writeAuthError(w, r, err)
		return
	}
	session, err := h.service.Exchange(r.Context(), ExchangeInput{
		IDToken:   request.IDToken,
		Client:    request.Client,
		UserAgent: r.UserAgent(),
		IPAddress: httpx.ClientIP(r),
	})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session.newTokenResponse())
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	httpx.SetNoStore(w)
	var request RefreshRequest
	if err := httpx.Decode(w, r, &request); err != nil {
		writeAuthError(w, r, err)
		return
	}
	session, err := h.service.Refresh(r.Context(), RefreshInput{
		RefreshToken: request.RefreshToken,
		UserAgent:    r.UserAgent(),
		IPAddress:    httpx.ClientIP(r),
	})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session.newTokenResponse())
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	httpx.SetNoStore(w)
	var request LogoutRequest
	if err := httpx.Decode(w, r, &request); err != nil {
		writeAuthError(w, r, err)
		return
	}
	if err := h.service.Logout(r.Context(), request.RefreshToken); err != nil {
		writeAuthError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) jwks(w http.ResponseWriter, r *http.Request) {
	keys, err := h.service.JWKS(r.Context())
	if err != nil {
		httpx.SetNoStore(w)
		writeAuthError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", jwksCacheControl)
	httpx.WriteJSON(w, http.StatusOK, jwksResponse{Keys: keys})
}

func (h *Handler) logoutAll(w http.ResponseWriter, r *http.Request) {
	httpx.SetNoStore(w)
	accountID, ok := accountIDFromPrincipal(r)
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("Authentication is required."))
		return
	}
	if err := h.service.LogoutAll(r.Context(), accountID); err != nil {
		writeAuthError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) linkIdentity(w http.ResponseWriter, r *http.Request) {
	httpx.SetNoStore(w)
	accountID, ok := accountIDFromPrincipal(r)
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("Authentication is required."))
		return
	}
	var request IdentityLinkRequest
	if err := httpx.Decode(w, r, &request); err != nil {
		writeAuthError(w, r, err)
		return
	}
	identity, err := h.service.LinkIdentity(r.Context(), accountID, request.IDToken)
	if err != nil {
		writeAuthError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, identity.newIdentityResponse())
}

func (h *Handler) unlinkIdentity(w http.ResponseWriter, r *http.Request) {
	httpx.SetNoStore(w)
	accountID, ok := accountIDFromPrincipal(r)
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("Authentication is required."))
		return
	}
	identityID, err := uuid.Parse(chi.URLParam(r, "identityId"))
	if err != nil {
		httpx.WriteError(w, r, apperrors.BadRequest(apperrors.CodeValidationFailed, "Identity id must be a valid UUID."))
		return
	}
	if err := h.service.UnlinkIdentity(r.Context(), accountID, identityID); err != nil {
		writeAuthError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func accountIDFromPrincipal(r *http.Request) (uuid.UUID, bool) {
	principal, ok := PrincipalFrom(r.Context())
	if !ok {
		return uuid.Nil, false
	}
	accountID := principal.AccountID
	if accountID == uuid.Nil && principal.ActorType == ActorAccount {
		accountID, _ = uuid.Parse(principal.ActorID)
	}
	return accountID, accountID != uuid.Nil
}

func writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	var rateLimited *RateLimitedError
	switch {
	case errors.As(err, &rateLimited):
		retryAfter := int(math.Ceil(rateLimited.RetryAfter.Seconds()))
		if retryAfter < 1 {
			retryAfter = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		httpx.WriteError(w, r, apperrors.RateLimited("Too many authentication attempts. Retry later."))
	case errors.Is(err, ErrAuthenticationFailed):
		httpx.WriteError(w, r, apperrors.Unauthorized("Authentication failed."))
	case errors.Is(err, ErrIdentityLinkRequired):
		httpx.WriteError(w, r, apperrors.Conflict("identity_link_required", "This sign-in must be linked to an existing account."))
	case errors.Is(err, ErrIdentityConflict):
		httpx.WriteError(w, r, apperrors.Conflict("identity_conflict", "This sign-in is already linked."))
	case errors.Is(err, ErrLastIdentity):
		httpx.WriteError(w, r, apperrors.Conflict("last_identity", "At least one sign-in method must remain linked."))
	case errors.Is(err, ErrIdentityNotFound):
		httpx.WriteError(w, r, apperrors.NotFound(apperrors.CodeNotFound, "Identity not found."))
	case errors.Is(err, ErrClientForbidden):
		httpx.WriteError(w, r, apperrors.Forbidden("Authentication is not permitted for this client."))
	case errors.Is(err, ErrFirebaseUnavailable),
		errors.Is(err, ErrRateLimiterUnavailable),
		errors.Is(err, ErrSessionUnavailable):
		w.Header().Set("Retry-After", "30")
		httpx.WriteError(w, r, apperrors.AuthUnavailable("Authentication is temporarily unavailable."))
	default:
		if _, ok := apperrors.As(err); ok {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteError(w, r, apperrors.Internal(err))
	}
}
