package retry

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"time"
)

var (
	tokenKey = []byte("devproxy-retry-secret-key-32bytes!")
	ErrInvalidToken = errors.New("quic retry: invalid token")
)

func GenerateToken(clientIP string, origDestConnID []byte, expiry time.Duration) ([]byte, error) {
	block, err := aes.NewCipher(tokenKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(expiry).UnixNano()
	plaintext := make([]byte, 8 + len(origDestConnID))
	putUint64(plaintext[0:8], uint64(expiresAt))
	copy(plaintext[8:], origDestConnID)

	ciphertext := aead.Seal(nonce, nonce, plaintext, []byte(clientIP))
	return ciphertext, nil
}

func ValidateToken(clientIP string, token []byte) ([]byte, error) {
	block, err := aes.NewCipher(tokenKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(token) < aead.NonceSize() {
		return nil, ErrInvalidToken
	}

	nonce := token[:aead.NonceSize()]
	ciphertext := token[aead.NonceSize():]

	plaintext, err := aead.Open(nil, nonce, ciphertext, []byte(clientIP))
	if err != nil {
		return nil, ErrInvalidToken
	}

	if len(plaintext) < 8 {
		return nil, ErrInvalidToken
	}

	expiresAt := int64(getUint64(plaintext[0:8]))
	if time.Now().UnixNano() > expiresAt {
		return nil, ErrInvalidToken
	}

	origDestConnID := make([]byte, len(plaintext)-8)
	copy(origDestConnID, plaintext[8:])
	return origDestConnID, nil
}

func putUint64(b []byte, v uint64) {
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
}

func getUint64(b []byte) uint64 {
	return uint64(b[0])<<56 |
		uint64(b[1])<<48 |
		uint64(b[2])<<40 |
		uint64(b[3])<<32 |
		uint64(b[4])<<24 |
		uint64(b[5])<<16 |
		uint64(b[6])<<8 |
		uint64(b[7])
}
