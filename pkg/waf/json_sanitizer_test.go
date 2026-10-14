package waf

import (
	"bytes"
	"testing"
)

func TestSanitizeStream(t *testing.T) {
	sanitizer := NewSanitizer()
	input := []byte(`{"user": "admin", "password": "secret123", "email": "test@example.com"}`)
	sensitive := [][]byte{[]byte("\"password\"")}

	output := sanitizer.SanitizeStream(input, sensitive)

	if bytes.Contains(output, []byte("secret123")) {
		t.Error("Failed to redact sensitive field")
	}

	if !bytes.Contains(output, []byte("REDACTED")) {
		t.Error("Redaction marker missing")
	}

	if sanitizer.RedactionCount.Load() != 1 {
		t.Errorf("Expected 1 redaction, got %d", sanitizer.RedactionCount.Load())
	}
}
