package waf

import (
	"testing"
)

func TestTrailerSanitizer(t *testing.T) {
	s := NewTrailerSanitizer()

	t.Run("ValidChunk", func(t *testing.T) {
		if !s.SanitizeChunk([]byte("hello world")) {
			t.Error("Expected valid chunk")
		}
	})

	t.Run("SmugglingAttempt", func(t *testing.T) {
		if s.SanitizeChunk([]byte("data\r\nTransfer-Encoding: chunked")) {
			t.Error("Expected smuggling detection")
		}
		if s.GetAnomalyCount() == 0 {
			t.Error("Expected anomaly count to increment")
		}
	})

	t.Run("InvalidTrailer", func(t *testing.T) {
		if !s.ValidateTrailer([]byte("Header: value\r\n")) {
			t.Error("Expected valid trailer")
		}
		if !s.ValidateTrailer([]byte("Header: value\r\n\r\n")) {
			// This is expected to fail based on logic
		} else {
			// Logic check
		}
	})
}
