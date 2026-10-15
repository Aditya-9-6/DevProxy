package h2priority

import (
	"testing"
)

func TestScheduler_Operations(t *testing.T) {
	s := NewScheduler()

	s.AddNode(1, 0, 16)
	s.UpdateWeight(1, 32)

	s.mu.RLock()
	n, ok := s.nodes[1]
	s.mu.RUnlock()

	if !ok || n.Weight.Load() != 32 {
		t.Errorf("Expected weight 32, got %d", n.Weight.Load())
	}

	s.RemoveNode(1)
	if len(s.nodes) != 0 {
		t.Error("Node was not removed")
	}
}
