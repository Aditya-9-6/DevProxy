package tracker

import (
	"testing"
)

func TestStateMachine(t *testing.T) {
	sm := NewStateMachine(10)

	// Test initial state
	if sm.Get(0) != StateIdle {
		t.Errorf("Expected StateIdle, got %v", sm.Get(0))
	}

	// Test valid transition
	if !sm.Update(0, StateIdle, StateSynSent) {
		t.Error("Failed to transition from Idle to SynSent")
	}

	if sm.Get(0) != StateSynSent {
		t.Errorf("Expected StateSynSent, got %v", sm.Get(0))
	}

	// Test invalid transition (CAS failure)
	if sm.Update(0, StateIdle, StateEstablished) {
		t.Error("Should have failed to transition from Idle to Established when current is SynSent")
	}

	// Test reset
	sm.Reset(0)
	if sm.Get(0) != StateIdle {
		t.Error("Failed to reset to Idle")
	}
}
