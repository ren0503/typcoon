package kms

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"testing"

	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
)

type fakeKMSClient struct {
	key *rsa.PrivateKey
}

func (f *fakeKMSClient) GetPublicKey(context.Context, *awskms.GetPublicKeyInput, ...func(*awskms.Options)) (*awskms.GetPublicKeyOutput, error) {
	der, err := x509.MarshalPKIXPublicKey(&f.key.PublicKey)
	if err != nil {
		return nil, err
	}
	return &awskms.GetPublicKeyOutput{PublicKey: der}, nil
}

func (f *fakeKMSClient) Sign(_ context.Context, input *awskms.SignInput, _ ...func(*awskms.Options)) (*awskms.SignOutput, error) {
	signature, err := rsa.SignPKCS1v15(rand.Reader, f.key, crypto.SHA256, input.Message)
	if err != nil {
		return nil, err
	}
	return &awskms.SignOutput{Signature: signature}, nil
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
