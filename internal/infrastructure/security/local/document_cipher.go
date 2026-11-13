package local

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

const documentEnvelopeMagic = "TYC1"

// LocalDocumentCipher is a non-KMS AES-GCM document cipher for development and
// tests. Ciphertext carries a versioned envelope so plaintext can be rejected.
type LocalDocumentCipher struct {
	keyID string
	aead  cipher.AEAD
}

// NewLocalDocumentCipher builds a local cipher with the supplied 16/24/32-byte key.
func NewLocalDocumentCipher(keyID string, key []byte) (*LocalDocumentCipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if keyID == "" {
		keyID = "local-dev"
	}
	return &LocalDocumentCipher{keyID: keyID, aead: aead}, nil
}

// NewDevDocumentCipher builds a local cipher with a random ephemeral key.
func NewDevDocumentCipher() (*LocalDocumentCipher, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return NewLocalDocumentCipher("local-dev", key)
}

// Encrypt seals plaintext into the versioned local envelope.
func (c *LocalDocumentCipher) Encrypt(plaintext []byte) ([]byte, string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, "", err
	}
	sealed := c.aead.Seal(nil, nonce, plaintext, []byte(documentEnvelopeMagic))
	out := make([]byte, 0, len(documentEnvelopeMagic)+len(nonce)+len(sealed))
	out = append(out, documentEnvelopeMagic...)
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, c.keyID, nil
}

// Validate rejects anything that is not a structurally valid ciphertext envelope.
func (c *LocalDocumentCipher) Validate(ciphertext []byte) error {
	if len(ciphertext) < len(documentEnvelopeMagic)+c.aead.NonceSize()+c.aead.Overhead() {
		return errors.New("ciphertext is too short")
	}
	if string(ciphertext[:len(documentEnvelopeMagic)]) != documentEnvelopeMagic {
		return errors.New("ciphertext envelope is not recognised")
	}
	return nil
}
