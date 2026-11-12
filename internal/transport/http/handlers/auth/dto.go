package auth

import (
	"math"
	"time"

	"github.com/google/uuid"
)

const jwksCacheControl = "public, max-age=300"

// jwksResponse is the public JSON Web Key Set response.
type jwksResponse struct {
	Keys []JWK `json:"keys"`
}

// FirebaseExchangeRequest is the public exchange payload.
type FirebaseExchangeRequest struct {
	IDToken string `json:"firebase_id_token" validate:"required"`
	Client  string `json:"client" validate:"omitempty,oneof=guest_mobile fleet_console"`
}

// RefreshRequest is the public refresh payload.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// IdentityLinkRequest is the authenticated identity-link payload.
type IdentityLinkRequest struct {
	IDToken string `json:"firebase_id_token" validate:"required"`
}

// LogoutRequest is the public logout payload.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type identityResponse struct {
	ID        uuid.UUID `json:"id"`
	AccountID uuid.UUID `json:"account_id"`
	Provider  string    `json:"provider"`
	Email     *string   `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (identity Identity) newIdentityResponse() identityResponse {
	return identityResponse{
		ID:        identity.ID,
		AccountID: identity.AccountID,
		Provider:  identity.Provider,
		Email:     identity.Email,
		CreatedAt: identity.CreatedAt,
	}
}

type sessionResponse struct {
	SessionID        uuid.UUID `json:"session_id"`
	AccountID        uuid.UUID `json:"account_id"`
	Client           string    `json:"client"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type tokenResponse struct {
	TokenType    string          `json:"token_type"`
	AccessToken  string          `json:"access_token"`
	ExpiresIn    int64           `json:"expires_in"`
	RefreshToken string          `json:"refresh_token"`
	Session      sessionResponse `json:"session"`
}

func (session IssuedSession) newTokenResponse() tokenResponse {
	expiresIn := int64(math.Ceil(time.Until(session.AccessExpiresAt).Seconds()))
	if expiresIn < 1 {
		expiresIn = 1
	}
	if expiresIn > 900 {
		expiresIn = 900
	}
	return tokenResponse{
		TokenType:    "Bearer",
		AccessToken:  session.AccessToken,
		ExpiresIn:    expiresIn,
		RefreshToken: session.RefreshToken,
		Session: sessionResponse{
			SessionID:        session.SessionID,
			AccountID:        session.AccountID,
			Client:           session.Client,
			AccessExpiresAt:  session.AccessExpiresAt,
			RefreshExpiresAt: session.RefreshExpiresAt,
		},
	}
}
