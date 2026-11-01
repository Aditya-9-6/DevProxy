package session

import (
	"testing"
)

func TestMachine(t *testing.T) {
	m := NewMachine()
	if m.state != StateIdle {
		t.Fatalf("expected idle")
	}
}