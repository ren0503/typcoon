package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type fixedClock struct {
	now time.Time
}

func (c *fixedClock) Now() time.Time {
	return c.now
}

func newTokenIssuer(t *testing.T) (*AccessTokenIssuer, *fixedClock) {
	t.Helper()
	signer := newTestSigner(t, "token-test")
	keys, err := newTestKeySet(t, "token-test", signer)
	if err != nil {
		t.Fatalf("NewStaticKeySet: %v", err)
	}
	clock := &fixedClock{now: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	issuer, err := NewAccessTokenIssuer(keys, "https://issuer.example", "tycoon-api", 15*time.Minute, time.Minute, clock)
	if err != nil {
		t.Fatalf("NewAccessTokenIssuer: %v", err)
	}
	return issuer, clock
}

func TestAccessTokenIssueAndValidate(t *testing.T) {
	issuer, clock := newTokenIssuer(t)
	accountID := uuid.New()
	raw, expiresAt, err := issuer.Issue(context.Background(), accountID, ClientGuestMobile)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if !expiresAt.Equal(clock.now.Add(15 * time.Minute)) {
		t.Fatalf("expiresAt = %v", expiresAt)
	}
	claims, err := issuer.Validate(context.Background(), raw)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if claims.Subject != accountID.String() || claims.Type != "access" || claims.Client != ClientGuestMobile {
		t.Fatalf("claims = %+v", claims)
	}
	if claims.ID == "" || claims.Issuer != "https://issuer.example" {
		t.Fatalf("registered claims = %+v", claims.RegisteredClaims)
	}
}

func TestAccessTokenRejectsInvalidClientAndLongTTL(t *testing.T) {
	issuer, _ := newTokenIssuer(t)
	if _, _, err := issuer.Issue(context.Background(), uuid.New(), "browser"); err == nil {
		t.Fatal("expected invalid client error")
	}
	signer := newTestSigner(t, "token-test")
	keys, err := newTestKeySet(t, "token-test", signer)
	if err != nil {
		t.Fatalf("NewStaticKeySet: %v", err)
	}
	if _, err := NewAccessTokenIssuer(keys, "issuer", "audience", MaxAccessTokenTTL+time.Second, time.Minute, nil); err == nil {
		t.Fatal("expected TTL rejection")
	}
}

func TestAccessTokenRejectsExpiredAndTampered(t *testing.T) {
	issuer, clock := newTokenIssuer(t)
	raw, _, err := issuer.Issue(context.Background(), uuid.New(), ClientGuestMobile)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	clock.now = clock.now.Add(16 * time.Minute)
	if _, err := issuer.Validate(context.Background(), raw); !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("expired token error = %v", err)
	}

	issuer, _ = newTokenIssuer(t)
	raw, _, err = issuer.Issue(context.Background(), uuid.New(), ClientGuestMobile)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if _, err := issuer.Validate(context.Background(), raw+"x"); !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("tampered token error = %v", err)
	}
}

func TestAccessTokenRejectsWrongAlgorithm(t *testing.T) {
	issuer, _ := newTokenIssuer(t)
	claims := AccessClaims{
		Type:   "access",
		Client: ClientGuestMobile,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://issuer.example",
			Subject:   uuid.NewString(),
			Audience:  jwt.ClaimStrings{"tycoon-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("not-the-key"))
	if err != nil {
		t.Fatalf("sign HS256 token: %v", err)
	}
	if _, err := issuer.Validate(context.Background(), raw); !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("wrong algorithm error = %v", err)
	}
}
