package proxyproto

import (
	"bytes"
	"net/netip"
	"testing"
)

func TestDecoder_ValidINET(t *testing.T) {
	raw := []byte{
		0x0D, 0x0A, 0x0D, 0x0A, 0x00, 0x0D, 0x0A, 0x51, 0x55, 0x49, 0x54, 0x0A,
		0x21, // version 2, PROXY command
		0x11, // AF_INET, STREAM
		0x00, 0x0C,
		192, 168, 1, 1,
		10, 0, 0, 1,
		0x04, 0xD2, // 1234
		0x00, 0x50, // 80
	}

	decoder := NewDecoder()
	hdr, n, err := decoder.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != len(raw) {
		Fatalf(t, "expected consumed bytes %d, got %d", len(raw), n)
	}

	expSrc := netip.MustParseAddrPort("192.168.1.1:1234")
	if hdr.Source != expSrc {
		t.Errorf("expected source %v, got %v", expSrc, hdr.Source)
	}
}

func TestDecoder_InvalidSignature(t *testing.T) {
	raw := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x21, 0x11, 0x00, 0x00,
	}
	decoder := NewDecoder()
	_, _, err := decoder.Decode(bytes.NewReader(raw))
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func Fatalf(t *testing.T, format string, args ...any) {
	t.Helper()
	t.Fatalf(format, args...)
}
