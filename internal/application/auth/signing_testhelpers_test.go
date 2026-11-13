package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"math/big"
	"testing"
)

type testSigner struct {
	kid string
	key *rsa.PrivateKey
}

func newTestSigner(t *testing.T, kid string) *testSigner {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return &testSigner{kid: kid, key: key}
}

func (s *testSigner) KID() string { return s.kid }

func (s *testSigner) PublicKey(context.Context) (*rsa.PublicKey, error) {
	return &s.key.PublicKey, nil
}

func (s *testSigner) SignJWT(_ context.Context, input []byte) ([]byte, error) {
	digest := sha256.Sum256(input)
	return rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, digest[:])
}

type testKeySet struct {
	active string
	all    map[string]*testSigner
}

func newTestKeySet(t *testing.T, active string, signers ...Signer) (*testKeySet, error) {
	t.Helper()
	set := &testKeySet{active: active, all: map[string]*testSigner{}}
	for _, signer := range signers {
		typed, ok := signer.(*testSigner)
		if !ok {
			t.Fatalf("unexpected signer type %T", signer)
		}
		set.all[typed.KID()] = typed
	}
	return set, nil
}

func (s *testKeySet) Active(context.Context) (Signer, error) {
	return s.all[s.active], nil
}

func (s *testKeySet) ByKID(_ context.Context, kid string) (Signer, error) {
	signer, ok := s.all[kid]
	if !ok {
		return nil, ErrInvalidAccessToken
	}
	return signer, nil
}

func (s *testKeySet) PublicJWKS(context.Context) ([]JWK, error) {
	keys := make([]JWK, 0, len(s.all))
	for kid, signer := range s.all {
		pub := &signer.key.PublicKey
		keys = append(keys, JWK{
			Kty: "RSA",
			KID: kid,
			Use: "sig",
			Alg: "RS256",
			N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		})
	}
	return keys, nil
}
