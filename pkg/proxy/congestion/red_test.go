package congestion

import (
	"testing"
)

func TestREDController(t *testing.T) {
	// min: 10, max: 20, maxProb: 1000 (100%)
	red := NewREDController(10, 20, 1000)

	// Below min: should never drop
	for i := 0; i < 9; i++ {
		red.Increment()
	}
	if red.ShouldDrop() {
		t.Error("Should not drop below min threshold")
	}

	// Above max: should always drop
	for i := 0; i < 15; i++ {
		red.Increment()
	}
	if !red.ShouldDrop() {
		t.Error("Should drop above max threshold")
	}

	// Test reset
	for i := 0; i < 25; i++ {
		red.Decrement()
	}
	if red.ShouldDrop() {
		t.Error("Should not drop after reset")
	}
}
