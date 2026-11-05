package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
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
