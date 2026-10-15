package waf

import (
	"testing"
)

func TestSSEMitigator(t *testing.T) {
	mitigator := NewSSEMitigator(5)
	hash := uint32(123)

	// Test normal operation
	for i := 0; i < 3; i++ {
		mitigator.RecordSaturation(hash)
	}

	if mitigator.ShouldDrop(hash) {
		t.Errorf("Expected no drop at saturation 3")
	}

	// Test threshold breach
	for i := 0; i < 3; i++ {
		mitigator.RecordSaturation(hash)
	}

	if !mitigator.ShouldDrop(hash) {
		t.Errorf("Expected drop at saturation 6")
	}

	// Test reset
	mitigator.ResetSaturation(hash)
	if mitigator.ShouldDrop(hash) {
		t.Errorf("Expected no drop after reset")
	}
}
