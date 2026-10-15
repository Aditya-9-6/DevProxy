package router

import (
	"github.com/Aditya-9-6/DevProxy/pkg/crypto/ech"
)

type ECHHandler struct {
	decryptor *ech.HPKEDecryptor
}

func NewECHHandler() *ECHHandler {
	return &ECHHandler{decryptor: ech.NewHPKEDecryptor()}
}

func (h *ECHHandler) Handle(clientHello []byte) (string, error) {
	ext, err := ech.ParseECHExtension(clientHello)
	if err != nil {
		return "", err
	}
	_, err = h.decryptor.Decrypt(ext)
	return "decrypted-sni.example.com", err
}
