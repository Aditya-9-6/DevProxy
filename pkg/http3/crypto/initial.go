package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
)

var (
	// QUIC v1 salt for Initial secret derivation (RFC 9001 section 5.2)
	quicV1Salt = []byte{0x38, 0x76, 0x2c, 0xf7, 0xf5, 0x59, 0x34, 0xb3, 0x4d, 0x17, 0x9a, 0xe6, 0xa4, 0xc8, 0x0c, 0xad, 0xcc, 0xbb, 0x7f, 0x0a}
	labeledSecretLabel = []byte("quic key")
	labeledIVLabel     = []byte("quic iv")
	labeledHPLabel     = []byte("quic hp")
)

// DeriveInitialSecrets derives client and server keys, IVs, and header protection keys
// from a destination connection ID following RFC 9001 HKDF-Extract and HKDF-Expand-Label.
func DeriveInitialSecrets(clientDstConnID []byte) (clientSecret, serverSecret []byte, err error) {
	if len(clientDstConnID) == 0 {
		return nil, nil, errors.New("empty destination connection ID")
	}

	// Initial Secret = HKDF-Extract(salt, client_dst_connection_id)
	initialSecret := hkdfExtract(quicV1Salt, clientDstConnID)

	// Client Initial Secret = HKDF-Expand-Label(initialSecret, "client in", "", Hash.Length)
	clientSecret = hkdfExpandLabel(initialSecret, []byte("client in"), 32)
	serverSecret = hkdfExpandLabel(initialSecret, []byte("server in"), 32)

	return clientSecret, serverSecret, nil
}

func hkdfExtract(salt, ikm []byte) []byte {
	h := hmac.New(sha256.New, salt)
	h.Write(ikm)
	return h.Sum(nil)
}

func hkdfExpandLabel(secret, label []byte, length int) []byte {
	// Simplified HKDF-Expand-Label per RFC 8445 / RFC 9001
	// info = length (2 bytes) + len(label)+5 (1 byte) + "tls13 " + label + len(context) (1 byte) + context
	context := []byte{}
	info := make([]byte, 0, 2+1+6+len(label)+1+len(context))
	
	info = append(info, byte(length>>8), byte(length))
	fullLabel := append([]byte("tls13 "), label...)
	info = append(info, byte(len(fullLabel)))
	info = append(info, fullLabel...)
	info = append(info, byte(len(context)))
	info = append(info, context...)

	h := hmac.New(sha256.New, secret)
	h.Write(info)
	out1 := h.Sum(nil)

	if len(out1) >= length {
		return out1[:length]
	}

	// If length > 32, expand further (not typical for initial secrets of 32 bytes)
	return out1
}

// DerivePacketKeys derives key, iv, and hp from secret
func DerivePacketKeys(secret []byte) (key, iv, hp []byte, err error) {
	if len(secret) < 32 {
		return nil, nil, nil, errors.New("secret too short")
	}

	key = hkdfExpandLabel(secret, labeledSecretLabel, 16)
	iv = hkdfExpandLabel(secret, labeledIVLabel, 12)
	hp = hkdfExpandLabel(secret, labeledHPLabel, 16)
	return key, iv, hp, nil
}
