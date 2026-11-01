package retry

import (
	"testing"
	"time"
)

func TestTokenGenerationAndValidation(t *testing.T) {
	ip := "127.0.0.1"
	odcid := []byte("original-dcid")

	token, err := GenerateToken(ip, odcid, time.Minute)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	validated, err := ValidateToken(ip, token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if string(validated) != string(odcid) {
		t.Fatalf("expected %s, got %s", odcid, validated)
	}
}

func TestIntegrityTag(t *testing.T) {
	odcid := []byte("original-dcid")
	pkt := []byte("some-quic-retry-packet-bytes")

	tag := GenerateIntegrityTag(odcid, pkt)
	if len(tag) != 16 {
		t.Fatalf("expected 16 byte tag, got %d", len(tag))
	}

	valid := VerifyIntegrityTag(odcid, pkt, tag)
	if !valid {
		t.Fatalf("expected tag to be valid")
	}
}
