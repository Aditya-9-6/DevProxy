package waf

import (
	"testing"
)

func TestHPACKGuard(t *testing.T) {
	guard := NewHPACKGuard(1024)

	// Test normal allocation
	if !guard.ValidateEntry(500) {
		t.Error("Expected validation to pass for 500 bytes")
	}

	if guard.GetCurrentSize() != 500 {
		t.Errorf("Expected size 500, got %d", guard.GetCurrentSize())
	}

	// Test overflow
	if guard.ValidateEntry(600) {
		t.Error("Expected validation to fail for 600 bytes (total 1100 > 1024)")
	}

	// Test release
	guard.ReleaseEntry(500)
	if guard.GetCurrentSize() != 0 {
		t.Errorf("Expected size 0 after release, got %d", guard.GetCurrentSize())
	}

	// Test boundary
	if !guard.ValidateEntry(1024) {
		t.Error("Expected validation to pass for exact limit")
	}
}
