package stream

import (
	"testing"
)

func TestSequencer(t *testing.T) {
	seq := NewSequencer(1)

	// Push out of order
	seq.Push(2, []byte("world"))
	seq.Push(1, []byte("hello"))

	// Pop 1
	seg1 := seq.PopReady()
	if seg1 == nil || seg1.Sequence != 1 {
		t.Fatal("Expected sequence 1")
	}

	// Pop 2
	seg2 := seq.PopReady()
	if seg2 == nil || seg2.Sequence != 2 {
		t.Fatal("Expected sequence 2")
	}
}
