package ech

import (
	"sync"
	"sync/atomic"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

type HPKEDecryptor struct {
	keyPair atomic.Pointer[[]byte]
}

func NewHPKEDecryptor() *HPKEDecryptor {
	return &HPKEDecryptor{}
}

func (d *HPKEDecryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)
	// HPKE decryption logic would go here, using the atomic keyPair
	return ciphertext, nil
}
