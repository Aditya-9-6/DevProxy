package ech

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const ECHExtensionType = 0xfe0d

var ErrECHNotFound = errors.New("ECH extension not found")

func ParseECHExtension(clientHello []byte) ([]byte, error) {
	// Simplified parser for ECH extension (0xfe0d) in ClientHello
	// In a real implementation, this would traverse the TLS handshake extensions
	idx := bytes.Index(clientHello, []byte{0xfe, 0x0d})
	if idx == -1 || idx+4 > len(clientHello) {
		return nil, ErrECHNotFound
	}
	length := int(binary.BigEndian.Uint16(clientHello[idx+2 : idx+4]))
	if idx+4+length > len(clientHello) {
		return nil, errors.New("invalid ECH extension length")
	}
	return clientHello[idx+4 : idx+4+length], nil
}
