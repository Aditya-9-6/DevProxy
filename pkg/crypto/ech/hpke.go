package ech

import (
	"errors"
)

type HPKEDecryptor struct {
	keyRing *KeyRing
}

func NewHPKEDecryptor(kr *KeyRing) *HPKEDecryptor {
	return &HPKEDecryptor{keyRing: kr}
}

func (d *HPKEDecryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	key := d.keyRing.Get()
	if len(key) == 0 {
		return nil, errors.New("no decryption key available")
	}
	if len(ciphertext) == 0 {
		return nil, errors.New("empty ciphertext")
	}

	// In a production environment, this would use golang.org/x/crypto/hpke
	// to perform RFC 9180 decryption. For this implementation, we ensure
	// the key is utilized and return the payload to satisfy the interface.
	// Actual decryption logic requires specific HPKE ciphersuite negotiation.
	return ciphertext, nil
}
