package waf

import (
	"testing"
)

func TestWebRTCInspector_Inspect(t *testing.T) {
	inspector := NewWebRTCInspector()

	// Test normal operation
	inspector.Inspect(1, 500)
	inspector.Inspect(1, 500)

	if inspector.GetStats(1) != 2 {
		t.Errorf("Expected count 2, got %d", inspector.GetStats(1))
	}

	// Test boundary conditions
	inspector.Inspect(999, 127000)
	if inspector.GetStats(999) != 1 {
		t.Errorf("Expected count 1 for index 999, got %d", inspector.GetStats(999))
	}

	// Test out of bounds
	inspector.Inspect(1001, 100)
	if inspector.GetStats(1001) != 0 {
		t.Error("Expected 0 for out of bounds index")
	}
}
