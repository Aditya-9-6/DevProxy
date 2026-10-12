package security

import (
	"testing"
)

func TestHTTP2StreamValidator_ValidSequence(t *testing.T) {
	val := NewHTTP2StreamValidator()

	// Valid start: HEADERS on stream 1
	anomaly, err := val.ValidateFrame(1, FrameHeaders, 0x4) // END_HEADERS
	if err != nil || anomaly != nil {
		t.Fatalf("unexpected anomaly on valid stream open: %v", anomaly)
	}

	// Data frame on open stream 1
	anomaly, err = val.ValidateFrame(1, FrameData, 0x1) // END_STREAM
	if err != nil || anomaly != nil {
		t.Fatalf("unexpected anomaly on valid data frame: %v", anomaly)
	}
}

func TestHTTP2StreamValidator_InvalidSequence(t *testing.T) {
	val := NewHTTP2StreamValidator()

	// Invalid start: DATA frame on unopened stream 3
	anomaly, err := val.ValidateFrame(3, FrameData, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if anomaly == nil {
		t.Fatalf("expected protocol anomaly when opening stream with DATA frame")
	}
}
