package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// MaxAccessTokenTTL is the hard upper bound for backend access tokens.
	MaxAccessTokenTTL = 15 * time.Minute

	ClientGuestMobile  = "guest_mobile"
	ClientFleetConsole = "fleet_console"

	defaultClockSkew = 60 * time.Second
)

var ErrInvalidAccessToken = errors.New("invalid access token")

// Clock supplies the current time for deterministic token tests.
type Clock interface {
	Now() time.Time
}

// SystemClock uses the host clock.
type SystemClock struct{}

// Now returns the current UTC time.
func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

// AccessClaims is the backend-owned access-token payload.
type AccessClaims struct {
	Type   string `json:"typ"`
	Client string `json:"client"`
	jwt.RegisteredClaims
}

// AccessTokenIssuer signs and validates backend access tokens.
type AccessTokenIssuer struct {
	keys     KeySet
	issuer   string
	audience string
	ttl      time.Duration
	leeway   time.Duration
	clock    Clock
}

// NewAccessTokenIssuer builds an RS256 access-token issuer.
func NewAccessTokenIssuer(keys KeySet, issuer, audience string, ttl, leeway time.Duration, clock Clock) (*AccessTokenIssuer, error) {
	if keys == nil {
		return nil, errors.New("key set is required")
	}
	if issuer == "" || audience == "" {
		return nil, errors.New("token issuer and audience are required")
	}
	if ttl <= 0 || ttl > MaxAccessTokenTTL {
		return nil, fmt.Errorf("access token TTL must be between 1s and %s", MaxAccessTokenTTL)
	}
	if leeway < 0 {
		return nil, errors.New("access token clock skew must not be negative")
	}
	if leeway == 0 {
		leeway = defaultClockSkew
	}
	if clock == nil {
		clock = SystemClock{}
	}
	return &AccessTokenIssuer{
		keys:     keys,
		issuer:   issuer,
		audience: audience,
		ttl:      ttl,
		leeway:   leeway,
		clock:    clock,
	}, nil
}

// Issue creates a backend access token for the account and client class.
func (i *AccessTokenIssuer) Issue(ctx context.Context, accountID uuid.UUID, client string) (string, time.Time, error) {
	if accountID == uuid.Nil {
		return "", time.Time{}, errors.New("account id is required")
	}
	if !validClient(client) {
		return "", time.Time{}, errors.New("invalid auth client")
	}
	signer, err := i.keys.Active(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	now := i.clock.Now().UTC()
	expiresAt := now.Add(i.ttl)
	claims := AccessClaims{
		Type:   "access",
		Client: client,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   accountID.String(),
			Audience:  jwt.ClaimStrings{i.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}
	method := &signerSigningMethod{ctx: ctx, signer: signer}
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = signer.KID()
	signed, err := token.SignedString(nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expiresAt, nil
}

// Validate verifies an access token and returns its backend claims.
func (i *AccessTokenIssuer) Validate(ctx context.Context, raw string) (AccessClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(i.issuer),
		jwt.WithAudience(i.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(i.leeway),
		jwt.WithTimeFunc(i.clock.Now),
	)
	claims := &AccessClaims{}
	_, err := parser.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, ErrInvalidAccessToken
		}
		signer, err := i.keys.ByKID(ctx, kid)
		if err != nil {
			return nil, err
		}
		return signer.PublicKey(ctx)
	})
	if err != nil {
		return AccessClaims{}, fmt.Errorf("%w: %v", ErrInvalidAccessToken, err)
	}
	if claims.Type != "access" || !validClient(claims.Client) || claims.ID == "" {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	if _, err := uuid.Parse(claims.Subject); err != nil {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	if claims.ExpiresAt.Sub(claims.IssuedAt.Time) > MaxAccessTokenTTL {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	return *claims, nil
}

func validClient(client string) bool {
	return client == ClientGuestMobile || client == ClientFleetConsole
}

type signerSigningMethod struct {
	ctx    context.Context
	signer Signer
}

func (m *signerSigningMethod) Alg() string {
	return signingAlgorithm
}

func (m *signerSigningMethod) Sign(signingString string, _ any) ([]byte, error) {
	return m.signer.SignJWT(m.ctx, []byte(signingString))
}

func (m *signerSigningMethod) Verify(signingString string, signature []byte, key any) error {
	publicKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return errors.New("RS256 verification requires an RSA public key")
	}
	digest := sha256.Sum256([]byte(signingString))
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature)
}

var _ jwt.SigningMethod = (*signerSigningMethod)(nil)
