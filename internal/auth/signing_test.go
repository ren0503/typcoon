package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestLocalSignerSignsVerifiableJWT(t *testing.T) {
	signer, err := NewRandomLocalSigner("local-test")
	if err != nil {
		t.Fatalf("NewRandomLocalSigner: %v", err)
	}
	input := []byte("header.payload")
	signature, err := signer.SignJWT(context.Background(), input)
	if err != nil {
		t.Fatalf("SignJWT: %v", err)
	}
	publicKey, err := signer.PublicKey(context.Background())
	if err != nil {
		t.Fatalf("PublicKey: %v", err)
	}
	digest := sha256.Sum256(input)
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
}

func TestLoadLocalSignerFromPEM(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	data, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	path := filepath.Join(t.TempDir(), "signing.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: data}), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	signer, err := LoadLocalSigner("pem-test", path)
	if err != nil {
		t.Fatalf("LoadLocalSigner: %v", err)
	}
	if signer.KID() != "pem-test" {
		t.Fatalf("KID = %q", signer.KID())
	}
}

func TestStaticKeySetPublishesOnlyPublicJWKFields(t *testing.T) {
	active, err := NewRandomLocalSigner("active")
	if err != nil {
		t.Fatalf("active signer: %v", err)
	}
	retired, err := NewRandomLocalSigner("retired")
	if err != nil {
		t.Fatalf("retired signer: %v", err)
	}
	set, err := NewStaticKeySet("active", active, retired)
	if err != nil {
		t.Fatalf("NewStaticKeySet: %v", err)
	}
	got, err := set.Active(context.Background())
	if err != nil {
		t.Fatalf("Active: %v", err)
	}
	if got.KID() != "active" {
		t.Fatalf("active KID = %q", got.KID())
	}
	if _, err := set.ByKID(context.Background(), "retired"); err != nil {
		t.Fatalf("ByKID: %v", err)
	}
	keys, err := set.PublicJWKS(context.Background())
	if err != nil {
		t.Fatalf("PublicJWKS: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("keys = %d, want 2", len(keys))
	}
	for _, key := range keys {
		if key.Kty != "RSA" || key.Use != "sig" || key.Alg != "RS256" || key.KID == "" {
			t.Fatalf("unexpected JWK: %+v", key)
		}
		if _, err := base64.RawURLEncoding.DecodeString(key.N); err != nil {
			t.Fatalf("decode modulus: %v", err)
		}
		if _, err := base64.RawURLEncoding.DecodeString(key.E); err != nil {
			t.Fatalf("decode exponent: %v", err)
		}
	}
}

type fakeKMSClient struct {
	key *rsa.PrivateKey
}

func (f *fakeKMSClient) GetPublicKey(context.Context, *kms.GetPublicKeyInput, ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error) {
	der, err := x509.MarshalPKIXPublicKey(&f.key.PublicKey)
	if err != nil {
		return nil, err
	}
	return &kms.GetPublicKeyOutput{PublicKey: der}, nil
}

func (f *fakeKMSClient) Sign(_ context.Context, input *kms.SignInput, _ ...func(*kms.Options)) (*kms.SignOutput, error) {
	signature, err := rsa.SignPKCS1v15(rand.Reader, f.key, crypto.SHA256, input.Message)
	if err != nil {
		return nil, err
	}
	return &kms.SignOutput{Signature: signature}, nil
}

func TestKMSSignerSignsAndImplementsCryptoSigner(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	client := &fakeKMSClient{key: key}
	signer, err := NewKMSSigner(context.Background(), client, "kms-test", "arn:aws:kms:test:key/test")
	if err != nil {
		t.Fatalf("NewKMSSigner: %v", err)
	}
	input := []byte("header.payload")
	signature, err := signer.SignJWT(context.Background(), input)
	if err != nil {
		t.Fatalf("SignJWT: %v", err)
	}
	digest := sha256.Sum256(input)
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], signature); err != nil {
		t.Fatalf("verify KMS signature: %v", err)
	}
	if signer.Public() == nil {
		t.Fatal("crypto.Signer Public() returned nil")
	}
}

func TestStaticKeySetRotationOverlapVerifiesOldTokens(t *testing.T) {
	clock := &fixedClock{now: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	oldSigner, err := NewRandomLocalSigner("old-key")
	if err != nil {
		t.Fatalf("old signer: %v", err)
	}
	newSigner, err := NewRandomLocalSigner("new-key")
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	oldSet, err := NewStaticKeySet("old-key", oldSigner)
	if err != nil {
		t.Fatalf("old key set: %v", err)
	}
	oldIssuer, err := NewAccessTokenIssuer(oldSet, "test-issuer", "test-audience", 15*time.Minute, time.Minute, clock)
	if err != nil {
		t.Fatalf("old issuer: %v", err)
	}
	accountID := uuid.New()
	oldToken, _, err := oldIssuer.Issue(context.Background(), accountID, ClientGuestMobile)
	if err != nil {
		t.Fatalf("issue old token: %v", err)
	}

	overlapSet, err := NewStaticKeySet("new-key", oldSigner, newSigner)
	if err != nil {
		t.Fatalf("overlap key set: %v", err)
	}
	overlapIssuer, err := NewAccessTokenIssuer(overlapSet, "test-issuer", "test-audience", 15*time.Minute, time.Minute, clock)
	if err != nil {
		t.Fatalf("overlap issuer: %v", err)
	}
	if _, err := overlapIssuer.Validate(context.Background(), oldToken); err != nil {
		t.Fatalf("old token did not verify during overlap: %v", err)
	}

	newToken, _, err := overlapIssuer.Issue(context.Background(), accountID, ClientGuestMobile)
	if err != nil {
		t.Fatalf("issue new token: %v", err)
	}
	parsed, _, err := jwt.NewParser().ParseUnverified(newToken, &AccessClaims{})
	if err != nil {
		t.Fatalf("parse new token: %v", err)
	}
	if kid, _ := parsed.Header["kid"].(string); kid != "new-key" {
		t.Fatalf("new token kid = %q, want new-key", kid)
	}
	if _, err := overlapIssuer.Validate(context.Background(), newToken); err != nil {
		t.Fatalf("new token did not verify: %v", err)
	}

	retiredSet, err := NewStaticKeySet("new-key", newSigner)
	if err != nil {
		t.Fatalf("retired key set: %v", err)
	}
	retiredIssuer, err := NewAccessTokenIssuer(retiredSet, "test-issuer", "test-audience", 15*time.Minute, time.Minute, clock)
	if err != nil {
		t.Fatalf("retired issuer: %v", err)
	}
	if _, err := retiredIssuer.Validate(context.Background(), oldToken); err == nil {
		t.Fatal("old token verified after its key was retired")
	}
}

func TestPublicJWKJSONContainsNoPrivateMaterial(t *testing.T) {
	signer, err := NewRandomLocalSigner("public-only")
	if err != nil {
		t.Fatalf("NewRandomLocalSigner: %v", err)
	}
	set, err := NewStaticKeySet("public-only", signer)
	if err != nil {
		t.Fatalf("NewStaticKeySet: %v", err)
	}
	keys, err := set.PublicJWKS(context.Background())
	if err != nil {
		t.Fatalf("PublicJWKS: %v", err)
	}
	raw, err := json.Marshal(keys)
	if err != nil {
		t.Fatalf("marshal keys: %v", err)
	}
	for _, forbidden := range []string{`"d"`, `"p"`, `"q"`, `"dp"`, `"dq"`, `"qi"`} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("JWKS leaked private field %s: %s", forbidden, raw)
		}
	}
}
