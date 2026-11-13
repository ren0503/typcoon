package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"
)

func TestOpaqueRefreshTokenGenerationAndHashing(t *testing.T) {
	first, firstHash, err := NewOpaqueRefreshToken()
	if err != nil {
		t.Fatalf("NewOpaqueRefreshToken: %v", err)
	}
	second, secondHash, err := NewOpaqueRefreshToken()
	if err != nil {
		t.Fatalf("NewOpaqueRefreshToken: %v", err)
	}
	if first == second {
		t.Fatal("refresh tokens were not unique")
	}
	raw, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatalf("decode refresh token: %v", err)
	}
	if len(raw) != refreshTokenBytes {
		t.Fatalf("raw token length = %d", len(raw))
	}
	expected := sha256.Sum256([]byte(first))
	if !bytes.Equal(firstHash, expected[:]) {
		t.Fatal("refresh token hash mismatch")
	}
	if bytes.Contains(firstHash, []byte(first)) || bytes.Contains(secondHash, []byte(second)) {
		t.Fatal("raw refresh token leaked into hash")
	}
	if !bytes.Equal(HashRefreshToken(first), firstHash) {
		t.Fatal("HashRefreshToken mismatch")
	}
}

func TestRefreshHelpers(t *testing.T) {
	firstFamily := NewRefreshFamilyID()
	secondFamily := NewRefreshFamilyID()
	if firstFamily == secondFamily {
		t.Fatal("refresh family ids were not unique")
	}
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	expiry, err := RefreshTokenExpiry(now, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("RefreshTokenExpiry: %v", err)
	}
	if !expiry.Equal(now.Add(30 * 24 * time.Hour)) {
		t.Fatalf("expiry = %v", expiry)
	}
	if _, err := RefreshTokenExpiry(now, 0); err == nil {
		t.Fatal("expected non-positive TTL rejection")
	}
}
