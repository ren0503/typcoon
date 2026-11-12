package auth

import (
	"context"
	"crypto/rsa"
	"time"
)

const (
	signingAlgorithm = "RS256"

	// RecommendedSigningKeyOverlap is the recommended rotation overlap.
	RecommendedSigningKeyOverlap = 24 * time.Hour
)

// Signer issues RS256 signatures and exposes the matching public key.
type Signer interface {
	KID() string
	PublicKey(ctx context.Context) (*rsa.PublicKey, error)
	SignJWT(ctx context.Context, signingInput []byte) ([]byte, error)
}

// JWK is the public JSON Web Key representation used by the JWKS endpoint.
type JWK struct {
	Kty string `json:"kty"`
	KID string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// KeySet resolves the active signing key and all public verification keys.
type KeySet interface {
	Active(ctx context.Context) (Signer, error)
	ByKID(ctx context.Context, kid string) (Signer, error)
	PublicJWKS(ctx context.Context) ([]JWK, error)
}
