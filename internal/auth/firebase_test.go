package auth

import (
	"context"
	"errors"
	"testing"

	firebaseauth "firebase.google.com/go/v4/auth"
)

type fakeFirebaseAuthClient struct {
	token *firebaseauth.Token
	err   error
}

func (f *fakeFirebaseAuthClient) VerifyIDTokenAndCheckRevoked(context.Context, string) (*firebaseauth.Token, error) {
	return f.token, f.err
}

func TestFirebaseVerifierMapsVerifiedClaims(t *testing.T) {
	verifier := &FirebaseVerifier{
		client: &fakeFirebaseAuthClient{token: &firebaseauth.Token{
			UID: "firebase-uid",
			Claims: map[string]interface{}{
				"email":          "guest@example.com",
				"email_verified": true,
				"name":           "Guest User",
			},
		}},
	}
	identity, err := verifier.Verify(context.Background(), "firebase-token")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if identity.UID != "firebase-uid" || identity.Email != "guest@example.com" || !identity.EmailVerified {
		t.Fatalf("identity = %+v", identity)
	}
	if identity.DisplayName != "Guest User" {
		t.Fatalf("DisplayName = %q", identity.DisplayName)
	}
}

func TestFirebaseVerifierIgnoresCustomAuthorizationClaims(t *testing.T) {
	verifier := &FirebaseVerifier{
		client: &fakeFirebaseAuthClient{token: &firebaseauth.Token{
			UID: "firebase-uid",
			Claims: map[string]interface{}{
				"email":          "admin@example.com",
				"email_verified": true,
				"name":           "Admin Looking User",
				"role":           "super_admin",
				"admin":          true,
				"tenant_id":      "00000000-0000-0000-0000-000000000001",
				"ship_ids":       []interface{}{"00000000-0000-0000-0000-000000000002"},
			},
		}},
	}
	identity, err := verifier.Verify(context.Background(), "firebase-token")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	want := VerifiedIdentity{
		UID:           "firebase-uid",
		Email:         "admin@example.com",
		EmailVerified: true,
		DisplayName:   "Admin Looking User",
	}
	if identity != want {
		t.Fatalf("verified identity = %+v, want only allowlisted fields %+v", identity, want)
	}
}

func TestFirebaseVerifierDropsUnverifiedEmail(t *testing.T) {
	verifier := &FirebaseVerifier{
		client: &fakeFirebaseAuthClient{token: &firebaseauth.Token{
			UID: "firebase-uid",
			Claims: map[string]interface{}{
				"email":          "guest@example.com",
				"email_verified": false,
			},
		}},
	}
	identity, err := verifier.Verify(context.Background(), "firebase-token")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if identity.Email != "" || identity.EmailVerified {
		t.Fatalf("unverified email was retained: %+v", identity)
	}
}

func TestFirebaseVerifierFailsClosed(t *testing.T) {
	verifier := &FirebaseVerifier{
		client: &fakeFirebaseAuthClient{err: errors.New("network unavailable")},
	}
	_, err := verifier.Verify(context.Background(), "firebase-token")
	if !errors.Is(err, ErrFirebaseUnavailable) {
		t.Fatalf("err = %v, want ErrFirebaseUnavailable", err)
	}
}

func TestFirebaseVerifierWithoutProjectIsUnavailable(t *testing.T) {
	verifier, err := NewFirebaseVerifier(context.Background(), FirebaseVerifierConfig{AppEnv: "development"})
	if err != nil {
		t.Fatalf("NewFirebaseVerifier: %v", err)
	}
	_, err = verifier.Verify(context.Background(), "firebase-token")
	if !errors.Is(err, ErrFirebaseUnavailable) {
		t.Fatalf("err = %v, want ErrFirebaseUnavailable", err)
	}
}
