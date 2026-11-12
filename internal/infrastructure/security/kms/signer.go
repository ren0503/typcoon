// Package kms implements AWS KMS-backed RS256 signing.
package kms

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"

	appauth "github.com/ren0503/tycoon/internal/application/auth"
)

const defaultKMSOperationTimeout = 5 * time.Second

// KMSClient is the subset of the AWS KMS API used for asymmetric signing.
type KMSClient interface {
	Sign(ctx context.Context, params *awskms.SignInput, optFns ...func(*awskms.Options)) (*awskms.SignOutput, error)
	GetPublicKey(ctx context.Context, params *awskms.GetPublicKeyInput, optFns ...func(*awskms.Options)) (*awskms.GetPublicKeyOutput, error)
}

// KMSSigner signs with an AWS KMS asymmetric key and implements crypto.Signer.
type KMSSigner struct {
	client    KMSClient
	kid       string
	keyARN    string
	publicKey *rsa.PublicKey
	timeout   time.Duration
}

// NewKMSSigner resolves the public key and returns a KMS-backed signer.
func NewKMSSigner(ctx context.Context, client KMSClient, kid, keyARN string) (*KMSSigner, error) {
	if client == nil {
		return nil, errors.New("KMS client is required")
	}
	if kid == "" || keyARN == "" {
		return nil, errors.New("KMS kid and key ARN are required")
	}
	signer := &KMSSigner{client: client, kid: kid, keyARN: keyARN, timeout: defaultKMSOperationTimeout}
	if err := signer.loadPublicKey(ctx); err != nil {
		return nil, err
	}
	return signer, nil
}

func (s *KMSSigner) loadPublicKey(ctx context.Context) error {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	output, err := s.client.GetPublicKey(callCtx, &awskms.GetPublicKeyInput{KeyId: aws.String(s.keyARN)})
	if err != nil {
		return fmt.Errorf("get KMS public key: %w", err)
	}
	parsed, err := x509.ParsePKIXPublicKey(output.PublicKey)
	if err != nil {
		return fmt.Errorf("parse KMS public key: %w", err)
	}
	publicKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return errors.New("KMS public key is not RSA")
	}
	s.publicKey = publicKey
	return nil
}

// KID returns the key identifier published in the JWT header.
func (s *KMSSigner) KID() string { return s.kid }

// PublicKey returns the cached KMS public key.
func (s *KMSSigner) PublicKey(context.Context) (*rsa.PublicKey, error) { return s.publicKey, nil }

// Public implements crypto.Signer with the cached public key.
func (s *KMSSigner) Public() crypto.PublicKey { return s.publicKey }

// Sign implements crypto.Signer for callers that already hashed the input.
func (s *KMSSigner) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if opts.HashFunc() != crypto.SHA256 {
		return nil, errors.New("KMS signer only supports SHA-256")
	}
	return s.signDigest(context.Background(), digest)
}

// SignJWT hashes the JWT signing input and delegates to KMS.
func (s *KMSSigner) SignJWT(ctx context.Context, signingInput []byte) ([]byte, error) {
	digest := sha256.Sum256(signingInput)
	return s.signDigest(ctx, digest[:])
}

func (s *KMSSigner) signDigest(ctx context.Context, digest []byte) ([]byte, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	output, err := s.client.Sign(callCtx, &awskms.SignInput{
		KeyId: aws.String(s.keyARN), Message: digest, MessageType: types.MessageTypeDigest,
		SigningAlgorithm: types.SigningAlgorithmSpecRsassaPkcs1v15Sha256,
	})
	if err != nil {
		return nil, fmt.Errorf("sign with KMS: %w", err)
	}
	return output.Signature, nil
}

var _ appauth.Signer = (*KMSSigner)(nil)
var _ crypto.Signer = (*KMSSigner)(nil)
