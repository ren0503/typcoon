package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

const defaultFirebaseTimeout = 5 * time.Second

var (
	// ErrInvalidFirebaseToken indicates a token that failed Firebase verification.
	ErrInvalidFirebaseToken = errors.New("invalid Firebase ID token")
	// ErrFirebaseUnavailable indicates that verification could not be completed safely.
	ErrFirebaseUnavailable = errors.New("firebase verification unavailable")
)

// VerifiedIdentity is the verified subset of a Firebase ID token used by auth.
type VerifiedIdentity struct {
	UID           string
	Email         string
	EmailVerified bool
	DisplayName   string
}

// Verifier verifies a Firebase ID Token and returns stable identity fields.
type Verifier interface {
	Verify(ctx context.Context, idToken string) (VerifiedIdentity, error)
}

// FirebaseVerifierConfig holds Firebase Admin SDK initialization settings.
type FirebaseVerifierConfig struct {
	ProjectID       string
	CredentialsFile string
	EmulatorHost    string
	AppEnv          string
	Timeout         time.Duration
}

type firebaseAuthClient interface {
	VerifyIDTokenAndCheckRevoked(ctx context.Context, idToken string) (*firebaseauth.Token, error)
}

// FirebaseVerifier verifies Firebase ID Tokens with revocation checking.
type FirebaseVerifier struct {
	client  firebaseAuthClient
}

// NewFirebaseVerifier builds the Firebase Admin SDK verifier. An optional
// emulator is accepted only in development or test.
func NewFirebaseVerifier(ctx context.Context, cfg FirebaseVerifierConfig) (*FirebaseVerifier, error) {
	if cfg.ProjectID == "" {
		return &FirebaseVerifier{}, nil
	}

	opts := make([]option.ClientOption, 0, 2)
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithAuthCredentialsFile(option.ServiceAccount, cfg.CredentialsFile))
	}
	if cfg.EmulatorHost != "" {
		if cfg.AppEnv != "development" && cfg.AppEnv != "test" {
			return nil, errors.New("firebase auth emulator is only allowed in development or test")
		}
		if err := os.Setenv("FIREBASE_AUTH_EMULATOR_HOST", cfg.EmulatorHost); err != nil {
			return nil, fmt.Errorf("configure Firebase Auth emulator: %w", err)
		}
		opts = append(opts, option.WithoutAuthentication())
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.ProjectID}, opts...)
	if err != nil {
		return nil, fmt.Errorf("initialize Firebase app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize Firebase Auth client: %w", err)
	}
	return &FirebaseVerifier{client: client}, nil
}

// Verify validates the token signature, project audience, and revocation state.
func (v *FirebaseVerifier) Verify(ctx context.Context, idToken string) (VerifiedIdentity, error) {
	if v.client == nil {
		return VerifiedIdentity{}, ErrFirebaseUnavailable
	}

	token, err := v.client.VerifyIDTokenAndCheckRevoked(ctx, idToken)
	if err != nil {
		if firebaseauth.IsIDTokenInvalid(err) {
			return VerifiedIdentity{}, ErrInvalidFirebaseToken
		}
		return VerifiedIdentity{}, fmt.Errorf("%w: %v", ErrFirebaseUnavailable, err)
	}
	if strings.TrimSpace(token.UID) == "" {
		return VerifiedIdentity{}, ErrInvalidFirebaseToken
	}

	identity := VerifiedIdentity{UID: token.UID}
	if email, ok := token.Claims["email"].(string); ok {
		if verified, _ := token.Claims["email_verified"].(bool); verified {
			identity.Email = email
			identity.EmailVerified = true
		}
	}
	if displayName, ok := token.Claims["name"].(string); ok {
		identity.DisplayName = displayName
	}
	return identity, nil
}

var _ Verifier = (*FirebaseVerifier)(nil)
