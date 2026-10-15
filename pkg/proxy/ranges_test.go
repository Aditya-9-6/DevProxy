package proxy

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseByteRanges(t *testing.T) {
	totalLength := int64(1000)

	coeffs, err := ParseByteRanges("bytes=0-499", totalLength)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(coeffs) != 1 || coeffs[0].Start != 0 || coeffs[0].End != 499 {
		t.Errorf("unexpected parsed range: %+v", coeffs)
	}

	// Multiple ranges
	coeffs, err = ParseByteRanges("bytes=0-199, 400-599", totalLength)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(coeffs) != 2 {
		t.Errorf("expected 2 ranges, got %d", len(coeffs))
	}

	// Suffix range
	coeffs, err = ParseByteRanges("bytes=-200", totalLength)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(coeffs) != 1 || coeffs[0].Start != 800 || coeffs[0].End != 999 {
		t.Errorf("unexpected suffix range: %+v", coeffs)
	}

	// Invalid range
	_, err = ParseByteRanges("invalid=0-100", totalLength)
	if err == nil {
		t.Error("expected error for invalid range unit, got nil")
	}
}

func TestMultipartBoundaryConcatenator(t *testing.T) {
	var buf bytes.Buffer
	concat := NewMultipartConcatenator(&buf, "text/plain")

	data1 := strings.NewReader("Hello ")
	data2 := strings.NewReader("World!")

	err := concat.WritePart(0, 5, 12, data1)
	if err != nil {
		t.Fatalf("failed to write part 1: %v", err)
	}

	err = concat.WritePart(6, 11, 12, data2)
	if err != nil {
		t.Fatalf("failed to write part 2: %v", err)
	}

	err = concat.Close()
	if err != nil {
		t.Fatalf("failed to close concatenator: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Content-Range: bytes 0-5/12") {
		t.Errorf("missing content range 1 in output: %s", out)
	}
	if !strings.Contains(out, "Hello ") {
		t.Errorf("missing body part 1 in output: %s", out)
	}
	if !strings.Contains(out, "World!") {
		t.Errorf("missing body part 2 in output: %s", out)
	}

	// Verify atomic stats
	if GlobalRangeStats.GetTotalChunks() < 2 {
		t.Errorf("expected at least 2 chunks recorded, got %d", GlobalRangeStats.GetTotalChunks())
	}
}
