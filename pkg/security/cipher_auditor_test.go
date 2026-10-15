package security

import (
	"testing"
)

func TestCipherAuditor(t *testing.T) {
	auditor := NewCipherAuditor()

	// Test recording
	auditor.Record(0x009C) // TLS_RSA_WITH_AES_128_GCM_SHA256

	if count := auditor.GetCount(0x009C); count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Test out of bounds
	auditor.Record(999)
	if count := auditor.GetCount(999); count != 0 {
		t.Errorf("expected count 0 for out of bounds, got %d", count)
	}
}
