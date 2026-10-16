package ech

import (
	"encoding/binary"
	"errors"
)

const ECHExtensionType = 0xfe0d

var ErrECHNotFound = errors.New("ECH extension not found")

// ParseECHExtension robustly parses a TLS ClientHello to find the ECH extension.
// It follows the TLS 1.3 handshake structure to avoid false positives.
func ParseECHExtension(clientHello []byte) ([]byte, error) {
	if len(clientHello) < 39 {
		return nil, errors.New("client hello too short")
	}

	// Skip Handshake Type (1), Length (3), Version (2), Random (32)
	offset := 38

	// Session ID: 1 byte length + data
	sidLen := int(clientHello[offset])
	offset += 1 + sidLen

	// Cipher Suites: 2 bytes length + data
	if offset+2 > len(clientHello) {
		return nil, errors.New("malformed cipher suites")
	}
	csLen := int(binary.BigEndian.Uint16(clientHello[offset:]))
	offset += 2 + csLen

	// Compression Methods: 1 byte length + data
	if offset+1 > len(clientHello) {
		return nil, errors.New("malformed compression methods")
	}
	cmLen := int(clientHello[offset])
	offset += 1 + cmLen

	// Extensions: 2 bytes length + data
	if offset+2 > len(clientHello) {
		return nil, ErrECHNotFound
	}
	extsLen := int(binary.BigEndian.Uint16(clientHello[offset:]))
	offset += 2

	end := offset + extsLen
	if end > len(clientHello) {
		end = len(clientHello)
	}

	// Iterate through extensions
	for offset+4 <= end {
		extType := binary.BigEndian.Uint16(clientHello[offset:])
		extLen := int(binary.BigEndian.Uint16(clientHello[offset+2:]))
		offset += 4

		if offset+extLen > end {
			return nil, errors.New("malformed extension length")
		}

		if extType == ECHExtensionType {
			return clientHello[offset : offset+extLen], nil
		}
		offset += extLen
	}

	return nil, ErrECHNotFound
}
