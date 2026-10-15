package router

import (
	"github.com/Aditya-9-6/DevProxy/pkg/crypto/ech"
)

type ECHHandler struct {
	decryptor *ech.HPKEDecryptor
	keyRing   *ech.KeyRing
}

func NewECHHandler() *ECHHandler {
	kr := &ech.KeyRing{}
	return &ECHHandler{
		keyRing:   kr,
		decryptor: ech.NewHPKEDecryptor(kr),
	}
}

func (h *ECHHandler) Handle(clientHello []byte) (string, error) {
	ext, err := ech.ParseECHExtension(clientHello)
	if err != nil {
		return "", err
	}
	plaintext, err := h.decryptor.Decrypt(ext)
	if err != nil {
		return "", err
	}
	// Return the decrypted plaintext (Inner ClientHello)
	return string(plaintext), nil
}
