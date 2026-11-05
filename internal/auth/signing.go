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
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

const signingAlgorithm = "RS256"

// Signer issues RS256 signatures and exposes the matching public key.
type Signer interface {
	KID() string
	PublicKey(ctx context.Context) (*rsa.PublicKey, error)
	SignJWT(ctx context.Context, signingInput []byte) ([]byte, error)
}

// JWK is the public JSON Web Key representation used by the JWKS endpoint.
type JWK struct {
	Kty string `json:"kty"`
	KID string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// KeySet resolves the active signing key and all public verification keys.
type KeySet interface {
	Active(ctx context.Context) (Signer, error)
	ByKID(ctx context.Context, kid string) (Signer, error)
	PublicJWKS(ctx context.Context) ([]JWK, error)
}

// StaticKeySet holds an active signer and any retired public keys during rotation.
type StaticKeySet struct {
	activeKID string
	signers   map[string]Signer
	order     []string
}

// NewStaticKeySet builds a key set with one active kid and optional retired keys.
func NewStaticKeySet(activeKID string, signers ...Signer) (*StaticKeySet, error) {
	if activeKID == "" {
		return nil, errors.New("active signing kid is required")
	}
	set := &StaticKeySet{
		activeKID: activeKID,
		signers:   make(map[string]Signer, len(signers)),
		order:     make([]string, 0, len(signers)),
	}
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
func (s *StaticKeySet) Active(context.Context) (Signer, error) {
	signer, ok := s.signers[s.activeKID]
	if !ok {
		return nil, fmt.Errorf("active signing kid %q not found", s.activeKID)
	}
	return signer, nil
}

// ByKID returns the signer for a token header kid.
func (s *StaticKeySet) ByKID(_ context.Context, kid string) (Signer, error) {
	signer, ok := s.signers[kid]
	if !ok {
		return nil, fmt.Errorf("signing kid %q not found", kid)
	}
	return signer, nil
}

// PublicJWKS returns public-only RSA keys in stable configured order.
func (s *StaticKeySet) PublicJWKS(ctx context.Context) ([]JWK, error) {
	keys := make([]JWK, 0, len(s.order))
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

// LoadLocalSigner loads a PEM PKCS#1 or PKCS#8 RSA key, or generates one when
// the path is empty.
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
func (s *LocalSigner) KID() string {
	return s.kid
}

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

// KMSClient is the subset of the AWS KMS API used for asymmetric signing.
type KMSClient interface {
	Sign(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error)
	GetPublicKey(ctx context.Context, params *kms.GetPublicKeyInput, optFns ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error)
}

// KMSSigner signs with an AWS KMS asymmetric key and implements crypto.Signer.
type KMSSigner struct {
	client    KMSClient
	kid       string
	keyARN    string
	publicKey *rsa.PublicKey
}

// NewKMSSigner resolves the public key and returns a KMS-backed signer.
func NewKMSSigner(ctx context.Context, client KMSClient, kid, keyARN string) (*KMSSigner, error) {
	if client == nil {
		return nil, errors.New("KMS client is required")
	}
	if kid == "" || keyARN == "" {
		return nil, errors.New("KMS kid and key ARN are required")
	}
	signer := &KMSSigner{client: client, kid: kid, keyARN: keyARN}
	if err := signer.loadPublicKey(ctx); err != nil {
		return nil, err
	}
	return signer, nil
}

func (s *KMSSigner) loadPublicKey(ctx context.Context) error {
	output, err := s.client.GetPublicKey(ctx, &kms.GetPublicKeyInput{KeyId: aws.String(s.keyARN)})
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
func (s *KMSSigner) KID() string {
	return s.kid
}

// PublicKey returns the cached KMS public key.
func (s *KMSSigner) PublicKey(context.Context) (*rsa.PublicKey, error) {
	return s.publicKey, nil
}

// Public implements crypto.Signer with the cached public key.
func (s *KMSSigner) Public() crypto.PublicKey {
	return s.publicKey
}

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
	output, err := s.client.Sign(ctx, &kms.SignInput{
		KeyId:            aws.String(s.keyARN),
		Message:          digest,
		MessageType:      types.MessageTypeDigest,
		SigningAlgorithm: types.SigningAlgorithmSpecRsassaPkcs1V15Sha256,
	})
	if err != nil {
		return nil, fmt.Errorf("sign with KMS: %w", err)
	}
	return output.Signature, nil
}

func rsaJWK(kid string, publicKey *rsa.PublicKey) JWK {
	return JWK{
		Kty: "RSA",
		KID: kid,
		Use: "sig",
		Alg: signingAlgorithm,
		N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
	}
}

var (
	_ Signer        = (*LocalSigner)(nil)
	_ Signer        = (*KMSSigner)(nil)
	_ crypto.Signer = (*KMSSigner)(nil)
)
