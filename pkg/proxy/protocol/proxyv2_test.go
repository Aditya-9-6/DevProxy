package protocol

import (
	"testing"
)

func TestParseProxyV2(t *testing.T) {
	// Minimal IPv4 header: 12 bytes sig + 1 ver/cmd + 1 fam + 2 len + 12 bytes addr
	buf := make([]byte, 28)
	copy(buf[0:12], []byte(ProxyV2Signature))
	buf[12] = 0x21 // Version 2, Proxy
	buf[13] = 0x11 // IPv4
	buf[14] = 0x00
	buf[15] = 0x0C // 12 bytes data

	// Src: 127.0.0.1, Dst: 192.168.1.1, Ports: 1234, 80
	copy(buf[16:20], []byte{127, 0, 0, 1})
	copy(buf[20:24], []byte{192, 168, 1, 1})
	buf[24], buf[25] = 0x04, 0xD2 // 1234
	buf[26], buf[27] = 0x00, 0x50 // 80

	h, n, err := ParseProxyV2(buf)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if n != 28 {
		t.Errorf("Expected length 28, got %d", n)
	}
	if h.SrcAddr.String() != "127.0.0.1" {
		t.Errorf("Expected 127.0.0.1, got %s", h.SrcAddr.String())
	}
}
