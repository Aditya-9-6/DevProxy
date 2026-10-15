package waf

import (
	"testing"
)

func TestParseDNSMessage(t *testing.T) {
	// Minimal valid DNS query (A record for example.com)
	data := []byte{
		0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 0x03, 'c', 'o', 'm', 0x00,
		0x00, 0x01, 0x00, 0x01,
	}

	err := ParseDNSMessage(data)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	if GlobalDoHMetrics.RecordTypes[1].Load() != 1 {
		t.Error("failed to record A record type")
	}
}
