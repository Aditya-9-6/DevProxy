package replay

import (
	"testing"
)

func TestSanitizer(t *testing.T) {
	s := NewSanitizer()
	body := []byte("password=12345&user=admin")
	sanitized := s.SanitizeBody(body)
	if string(sanitized) != "password=[REDACTED]&user=admin" {
		t.Errorf("Expected redaction, got %s", string(sanitized))
	}
}

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(1024)
	rb.Push("test")
	if rb.buffer[0] != "test" {
		t.Error("Failed to push to ring buffer")
	}
}
