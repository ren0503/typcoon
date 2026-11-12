// Package local implements in-process RSA signing for development and tests.
package local

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	appauth "github.com/ren0503/tycoon/internal/application/auth"
)

const signingAlgorithm = "RS256"

// RecommendedSigningKeyOverlap is the recommended rotation overlap.
const RecommendedSigningKeyOverlap = 24 * time.Hour

// StaticKeySet holds an active signer and any retired public keys during rotation.
type StaticKeySet struct {
	activeKID string
	signers   map[string]appauth.Signer
	order     []string
}

// NewStaticKeySet builds a key set with one active kid and optional retired keys.
func NewStaticKeySet(activeKID string, signers ...appauth.Signer) (*StaticKeySet, error) {
	if activeKID == "" {
		return nil, errors.New("active signing kid is required")
	}
	set := &StaticKeySet{activeKID: activeKID, signers: make(map[string]appauth.Signer, len(signers)), order: make([]string, 0, len(signers))}
	for _, signer := range signers {
		if signer == nil {
			return nil, errors.New("signer is required")
		}
		kid := signer.KID()
		if kid == "" {
			return nil, errors.New("signing kid is required")
		}
		if _, exists := set.signers[kid]; exists {
			return nil, fmt.Errorf("duplicate signing kid %q", kid)
		}
		set.signers[kid] = signer
		set.order = append(set.order, kid)
	}
	if _, ok := set.signers[activeKID]; !ok {
		return nil, fmt.Errorf("active signing kid %q has no signer", activeKID)
	}
	return set, nil
}

// Active returns the signer used for newly issued tokens.
func (s *StaticKeySet) Active(context.Context) (appauth.Signer, error) {
	signer, ok := s.signers[s.activeKID]
	if !ok {
		return nil, fmt.Errorf("active signing kid %q not found", s.activeKID)
	}
	return signer, nil
}

// ByKID returns the signer for a token header kid.
func (s *StaticKeySet) ByKID(_ context.Context, kid string) (appauth.Signer, error) {
	signer, ok := s.signers[kid]
	if !ok {
		return nil, fmt.Errorf("signing kid %q not found", kid)
	}
	return signer, nil
}

// PublicJWKS returns public-only RSA keys in stable configured order.
func (s *StaticKeySet) PublicJWKS(ctx context.Context) ([]appauth.JWK, error) {
	keys := make([]appauth.JWK, 0, len(s.order))
	for _, kid := range s.order {
		publicKey, err := s.signers[kid].PublicKey(ctx)
		if err != nil {
			return nil, err
		}
		keys = append(keys, rsaJWK(kid, publicKey))
	}
	return keys, nil
}

// LocalSigner signs tokens with an in-process RSA private key.
type LocalSigner struct {
	kid string
	key *rsa.PrivateKey
}

// NewLocalSigner wraps an existing RSA private key.
func NewLocalSigner(kid string, key *rsa.PrivateKey) (*LocalSigner, error) {
	if kid == "" {
		return nil, errors.New("signing kid is required")
	}
	if key == nil {
		return nil, errors.New("RSA private key is required")
	}
	return &LocalSigner{kid: kid, key: key}, nil
}

// NewRandomLocalSigner generates a development-only RSA key.
func NewRandomLocalSigner(kid string) (*LocalSigner, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate RSA signing key: %w", err)
	}
	return NewLocalSigner(kid, key)
}

// LoadLocalSigner loads a PEM PKCS#1 or PKCS#8 RSA key, or generates one when the path is empty.
func LoadLocalSigner(kid, path string) (*LocalSigner, error) {
	if path == "" {
		return NewRandomLocalSigner(kid)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read signing key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("signing key is not valid PEM")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return NewLocalSigner(kid, key)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse signing key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("signing key is not RSA")
	}
	return NewLocalSigner(kid, key)
}

// KID returns the key identifier published in the JWT header.
func (s *LocalSigner) KID() string { return s.kid }

// PublicKey returns the local RSA public key.
func (s *LocalSigner) PublicKey(context.Context) (*rsa.PublicKey, error) {
	return &s.key.PublicKey, nil
}

// SignJWT hashes the JWT signing input and applies RS256.
func (s *LocalSigner) SignJWT(_ context.Context, signingInput []byte) ([]byte, error) {
	digest := sha256.Sum256(signingInput)
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, digest[:])
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}
	return signature, nil
}

func rsaJWK(kid string, publicKey *rsa.PublicKey) appauth.JWK {
	return appauth.JWK{
		Kty: "RSA", KID: kid, Use: "sig", Alg: signingAlgorithm,
		N: base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
		E: base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
	}
}

var _ appauth.Signer = (*LocalSigner)(nil)
