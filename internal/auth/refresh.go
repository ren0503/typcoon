package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const refreshTokenBytes = 32

// NewOpaqueRefreshToken creates a 256-bit base64url credential and its SHA-256 hash.
func NewOpaqueRefreshToken() (string, []byte, error) {
	raw := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate refresh token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, HashRefreshToken(token), nil
}

// HashRefreshToken returns the storage hash for an opaque refresh token.
func HashRefreshToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// NewRefreshFamilyID creates a new independently revocable session family.
func NewRefreshFamilyID() uuid.UUID {
	return uuid.New()
}

// RefreshTokenExpiry calculates the refresh-token expiry from the given clock.
func RefreshTokenExpiry(now time.Time, ttl time.Duration) (time.Time, error) {
	if ttl <= 0 {
		return time.Time{}, errors.New("refresh token TTL must be positive")
	}
	return now.UTC().Add(ttl), nil
}
