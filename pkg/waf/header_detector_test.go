package waf

import (
	"testing"
)

func TestHeaderDetector(t *testing.T) {
	detector := NewHeaderDetector(10)
	vector := &ViolationVector{}

	// Test length overflow
	detector.Detect("X-Long-Header-Name", "value", vector)
	if !vector.Has(ViolationLengthOverflow) {
		t.Error("Expected ViolationLengthOverflow")
	}

	// Test forbidden chars
	vector2 := &ViolationVector{}
	detector.Detect("Bad:Key", "val", vector2)
	if !vector2.Has(ViolationForbiddenChar) {
		t.Error("Expected ViolationForbiddenChar")
	}

	// Test clean header
	vector3 := &ViolationVector{}
	detector.Detect("X-Valid", "val", vector3)
	if vector3.bits.Load() != 0 {
		t.Error("Expected no violations")
	}
}
