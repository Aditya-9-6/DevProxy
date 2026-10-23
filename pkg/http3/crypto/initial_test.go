package crypto

import (
	"testing"
)

func TestDeriveInitialSecrets(t *testing.T) {
	connID := []byte{0x83, 0x94, 0xc8, 0xf0, 0x3b, 0x2a, 0x11, 0xe5}
	cs, ss, err := DeriveInitialSecrets(connID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cs) != 32 || len(ss) != 32 {
		t.Errorf("expected 32-byte secrets, got client=%d, server=%d", len(cs), len(ss))
	}
}

func TestDerivePacketKeys(t *testing.T) {
	secret := make([]byte, 32)
	key, iv, hp, err := DerivePacketKeys(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 16 || len(iv) != 12 || len(hp) != 16 {
		t.Errorf("invalid key sizes: key=%d, iv=%d, hp=%d", len(key), len(iv), len(hp))
	}
}
