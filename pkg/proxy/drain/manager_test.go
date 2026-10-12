package drain

import (
	"testing"
)

func TestManager_Drain(t *testing.T) {
	m := NewManager()

	if !m.Register() {
		t.Error("Expected registration to succeed")
	}

	m.Drain()

	if m.Register() {
		t.Error("Expected registration to fail during drain")
	}

	m.Unregister()
	if m.ActiveCount() != 0 {
		t.Errorf("Expected 0 active connections, got %d", m.ActiveCount())
	}
}
