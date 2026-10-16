package tls

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

// SessionKeyRing holds the active AEAD cipher for ticket encryption.
type SessionKeyRing struct {
	AEAD  cipher.AEAD
	KeyID [16]byte
}

// NewSessionKeyRing generates a new key ring with a random AES-GCM key.
func NewSessionKeyRing() (*SessionKeyRing, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ring := &SessionKeyRing{AEAD: aead}
	_, _ = io.ReadFull(rand.Reader, ring.KeyID[:])
	return ring, nil
}
