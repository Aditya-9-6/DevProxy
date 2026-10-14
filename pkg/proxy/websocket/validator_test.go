package websocket

import "testing"

func TestFrameValidator(t *testing.T) {
	v := &FrameValidator{}
	if !v.ValidateMask([4]byte{1, 2, 3, 4}) {
		t.Error("Expected valid mask")
	}
	if v.ValidateMask([4]byte{0, 0, 0, 0}) {
		t.Error("Expected invalid mask")
	}
	if v.ViolationCount.Load() != 1 {
		t.Errorf("Expected 1 violation, got %d", v.ViolationCount.Load())
	}
}
