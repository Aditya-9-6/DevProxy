package ech

import (
	"encoding/binary"
	"testing"
)

func TestKeyRing_Rotate(t *testing.T) {
	kr := &KeyRing{}
	key := []byte("initial-key")
	kr.Rotate(key)

	// Verify copy
	key[0] = 'X'
	if string(kr.Get()) == "Xnitial-key" {
		t.Error("KeyRing did not copy the key during Rotate")
	}

	if string(kr.Get()) != "initial-key" {
		t.Errorf("Expected initial-key, got %s", string(kr.Get()))
	}
}

func TestParseECHExtension(t *testing.T) {
	// Build a mock ClientHello
	ch := make([]byte, 200)
	// Handshake prefix
	ch[38] = 0 // Session ID length
	// Cipher suites
	binary.BigEndian.PutUint16(ch[39:], 2)
	// Compression
	ch[43] = 0
	// Extensions length
	binary.BigEndian.PutUint16(ch[44:], 10)
	// Extension: ECH (0xfe0d)
	binary.BigEndian.PutUint16(ch[46:], ECHExtensionType)
	binary.BigEndian.PutUint16(ch[48:], 6)
	copy(ch[50:], "secret")

	payload, err := ParseECHExtension(ch[:56])
	if err != nil {
		t.Fatalf("Failed to parse ECH: %v", err)
	}
	if string(payload) != "secret" {
		t.Errorf("Expected 'secret', got %s", string(payload))
	}

	// Test malformed
	_, err = ParseECHExtension([]byte{0, 1, 2})
	if err == nil {
		t.Error("Expected error for short ClientHello")
	}
}

func TestHPKEDecryptor_Decrypt(t *testing.T) {
	kr := &KeyRing{}
	kr.Rotate([]byte("key"))
	dec := NewHPKEDecryptor(kr)

	plaintext, err := dec.Decrypt([]byte("ciphertext"))
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if string(plaintext) != "ciphertext" {
		t.Errorf("Expected ciphertext, got %s", string(plaintext))
	}

	kr.Rotate(nil)
	_, err = dec.Decrypt([]byte("data"))
	if err == nil {
		t.Error("Expected error with no key")
	}
}
