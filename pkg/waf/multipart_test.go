package waf

import (
	"bytes"
	"testing"
)

func TestMultipartNormalization(t *testing.T) {
	state := &MultipartState{}

	// Test normal boundary
	for i := 0; i < 3; i++ {
		if !state.NormalizeBoundary([]byte("boundary")) {
			t.Errorf("unexpected violation at depth %d", i)
		}
	}

	// Test violation
	state.boundaryDepth.Store(6)
	if state.NormalizeBoundary([]byte("nested")) {
		t.Error("failed to detect boundary nesting violation")
	}
}

func TestContentDispositionNormalization(t *testing.T) {
	input := []byte("form-data; name=\"file\"; filename=\"test.txt\r\nInjection: true\"")
	output := NormalizeContentDisposition(input)
	if bytes.Contains(output, []byte("\r")) || bytes.Contains(output, []byte("\n")) {
		t.Error("failed to strip CRLF from header")
	}
}
