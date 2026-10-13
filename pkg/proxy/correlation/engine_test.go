package correlation

import (
	"testing"
)

func TestCorrelationEngine(t *testing.T) {
	engine := NewEngine()

	// Test ID generation
	id1 := engine.NextID()
	id2 := engine.NextID()
	if id1 == id2 {
		t.Errorf("Expected unique IDs, got %d and %d", id1, id2)
	}

	// Test Pool acquisition
	entry := engine.Acquire()
	if entry == nil {
		t.Fatal("Failed to acquire entry")
	}
	if !entry.Active.Load() {
		t.Error("Entry should be active")
	}

	// Test Release
	engine.Release(entry)
	if entry.Active.Load() {
		t.Error("Entry should be inactive after release")
	}
}
