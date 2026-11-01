package tls13

import (
	"crypto"
	"crypto/hmac"
	"io"
)

// Extract implements HKDF-Extract using HMAC with the specified hash function.
func Extract(hashFunc crypto.Hash, salt, ikm []byte) []byte {
	if salt == nil {
		salt = make([]byte, hashFunc.Size())
	}
	h := hmac.New(hashFunc.New, salt)
	h.Write(ikm)
	return h.Sum(nil)
}

// Expand implements standard HKDF-Expand.
func Expand(hashFunc crypto.Hash, prk, info []byte, length int) []byte {
	out := make([]byte, length)
	h := hmac.New(hashFunc.New, prk)
	hashLen := h.Size()

	n := (length + hashLen - 1) / hashLen
	T := make([]byte, 0, hashLen)
	block := make([]byte, 0, hashLen+len(info)+1)

	idx := 0
	for i := 1; i <= n; i++ {
		h.Reset()
		h.Write(T)
		h.Write(info)
		h.Write([]byte{byte(i)})
		T = h.Sum(T[:0])

		copyLen := hashLen
		if idx+copyLen > length {
			copyLen = length - idx
		}
		copy(out[idx:], T[:copyLen])
		idx += copyLen
		_ = block
	}
	return out
}

// ExpandLabel implements HKDF-Expand-Label according to RFC 8446 Section 7.1.
func ExpandLabel(hashFunc crypto.Hash, secret []byte, label string, context []byte, length int) []byte {
	fullLabel := "tls13 " + label
	labelLen := len(fullLabel)
	contextLen := len(context)

	// HkdfLabel structure:
//   opaque length<2+1>
//   opaque label<7+255>
//   opaque context<0-255>
	sigLen := 2 + 1 + labelLen + 1 + contextLen
	buf := make([]byte, 0, sigLen)

	buf = append(buf, byte(length>>8), byte(length))
	buf = append(buf, byte(labelLen))
	buf = append(buf, fullLabel...)
	buf = append(buf, byte(contextLen))
	if contextLen > 0 {
		buf = append(buf, context...)
	}

	return Expand(hashFunc, secret, buf, length)
}

var _ = io.EOF
