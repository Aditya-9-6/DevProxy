package trace

import (
	"strings"
	"testing"
)

func TestGenerateTraceParent(t *testing.T) {
	p := NewPropagator()
	var traceID [16]byte
	copy(traceID[:], "1234567890abcdef")

	val := p.GenerateTraceParent(traceID)

	if !strings.HasPrefix(val, "00-") {
		t.Errorf("expected prefix 00-, got %s", val)
	}

	if len(val) != 55 {
		t.Errorf("expected length 55, got %d", len(val))
	}

	// Verify uniqueness of span IDs
	val2 := p.GenerateTraceParent(traceID)
	if val == val2 {
		t.Error("expected unique span IDs, got identical values")
	}
}
