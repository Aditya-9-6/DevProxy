package datagram

import (
	"bytes"
	"testing"
)

func TestDatagramStripper_NormalOperation(t *testing.T) {
	stripper := NewDatagramStripper(0.5)

	// Normal payload with zero padding
	raw := []byte("HTTP/3 QUIC Datagram Payload Test")
	cleaned, err := stripper.ProcessDatagram(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(cleaned, raw) {
		t.Errorf("expected %q, got %q", raw, cleaned)
	}
}

func TestDatagramStripper_PaddingStripping(t *testing.T) {
	stripper := NewDatagramStripper(0.5)

	// Payload followed by minimal trailing zero padding
	raw := append([]byte("ValidPayload"), 0x00, 0x00, 0x00, 0x00)
	cleaned, err := stripper.ProcessDatagram(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []byte("ValidPayload")
	if !bytes.Equal(cleaned, expected) {
		t.Errorf("expected %q, got %q", expected, cleaned)
	}
}

func TestDatagramStripper_AmplificationMitigation(t *testing.T) {
	stripper := NewDatagramStripper(0.5)

	// Datagram consisting almost entirely of zero padding (>80% padding ratio)
	raw := append([]byte("a"), make([]byte, 20)...)
	_, err := stripper.ProcessDatagram(raw)
	if err != ErrExcessiveAmplification {
		t.Errorf("expected ErrExcessiveAmplification, got %v", err)
	}
}
