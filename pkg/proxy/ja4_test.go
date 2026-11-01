package proxy

import (
	"testing"
)

func TestGenerateJA4(t *testing.T) {
	ch := &ClientHelloRecord{
		Version:      0x0304,
		CipherSuites: []uint16{0x1301, 0x1302, 0x0a0a},
		Extensions:   []uint16{0x0000, 0x0010, 0x002b, 0x0a0a},
		ServerName:   "example.com",
		ALPN:         "h2",
	}

	ja4 := GenerateJA4(ch)
	if len(ja4) == 0 {
		t.Fatal("expected non-empty JA4 fingerprint")
	}

	// Check structure: part1_hashciphers_hashextensions
	if ja4[0] != 't' {
		t.Errorf("expected protocol 't', got %c", ja4[0])
	}
}

func TestGenerateJA4S(t *testing.T) {
	sh := &ServerHelloRecord{
		Version:     0x0304,
		CipherSuite: 0x1301,
		Extensions:  []uint16{0x0010, 0x002b},
		ALPN:        "h2",
	}

	ja4s := GenerateJA4S(sh)
	if len(ja4s) == 0 {
		t.Fatal("expected non-empty JA4S fingerprint")
	}

	if ja4s[0] != 't' {
		t.Errorf("expected protocol 't', got %c", ja4s[0])
	}
}
