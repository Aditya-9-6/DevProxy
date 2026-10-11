package security

import (
	"testing"
)

func TestStreamReassembler(t *testing.T) {
	r := NewStreamReassembler(0)

	// Push out of order
	r.Push(5, []byte("world"))
	ready := r.Push(0, []byte("hello"))

	if len(ready) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(ready))
	}
	if string(ready[0]) != "hello" || string(ready[1]) != "world" {
		t.Errorf("reassembly failed: %s %s", ready[0], ready[1])
	}
}
