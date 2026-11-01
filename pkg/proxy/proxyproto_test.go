package proxy

import (
	"bytes"
	"net/netip"
	"testing"
)

func TestProxyProtoHandler_TrustedGate(t *testing.T) {
	prefix := netip.MustParsePrefix("192.168.1.0/24")
	handler := NewProxyProtoHandler([]netip.Prefix{prefix})

	trustedIP := netip.MustParseAddr("192.168.1.50")
	untrustedIP := netip.MustParseAddr("10.0.0.1")

	if !handler.IsTrusted(trustedIP) {
		t.Errorf("expected %v to be trusted", trustedIP)
	}
	if handler.IsTrusted(untrustedIP) {
		t.Errorf("expected %v to be untrusted", untrustedIP)
	}

	raw := []byte{
		0x0D, 0x0A, 0x0D, 0x0A, 0x00, 0x0D, 0x0A, 0x51, 0x55, 0x49, 0x54, 0x0A,
		0x21, 0x11, 0x00, 0x0C,
		192, 168, 1, 1,
		10, 0, 0, 1,
		0x04, 0xD2,
		0x00, 0x50,
	}

	_, err := handler.ParseHeader(bytes.NewReader(raw), untrustedIP)
	if err == nil {
		t.Error("expected error for untrusted IP header parse, got nil")
	}

	hdr, err := handler.ParseHeader(bytes.NewReader(raw), trustedIP)
	if err != nil {
		t.Fatalf("unexpected error for trusted IP: %v", err)
	}
	if hdr.Command != 0x01 {
		t.Errorf("expected command 0x01, got %d", hdr.Command)
	}
}
