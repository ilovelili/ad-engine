package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

type CredentialSealer struct {
	key []byte
}

func NewCredentialSealer(secret string) *CredentialSealer {
	sum := sha256.Sum256([]byte(secret))
	return &CredentialSealer{key: sum[:]}
}

func (s *CredentialSealer) Seal(value string) ([]byte, []byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}

	return nonce, gcm.Seal(nil, nonce, []byte(value), nil), nil
}
