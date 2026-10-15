package h2priority

import (
	"sync"
	"testing"
)

func TestScheduler_Operations(t *testing.T) {
	t.Parallel()
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

func TestScheduler_HierarchyAndReparenting(t *testing.T) {
	t.Parallel()
	s := NewScheduler()
	s.AddNode(1, 0, 16)
	s.AddNode(2, 1, 32)
	s.AddNode(3, 2, 64)

	s.mu.RLock()
	if len(s.nodes[1].Children) != 1 || s.nodes[1].Children[2] == nil {
		t.Error("Node 2 should be child of node 1")
	}
	s.mu.RUnlock()

	// Removing node 2 should reparent node 3 to node 1
	s.RemoveNode(2)

	s.mu.RLock()
	if s.nodes[3].ParentID != 1 {
		t.Errorf("Expected node 3 parent to be reparented to 1, got %d", s.nodes[3].ParentID)
	}
	s.mu.RUnlock()
}

func TestScheduler_ConcurrentStress(t *testing.T) {
	t.Parallel()
	s := NewScheduler()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id uint32) {
			defer wg.Done()
			s.AddNode(id, 0, uint64(id*10))
			s.UpdateWeight(id, uint64(id*20))
			s.RemoveNode(id)
		}(uint32(i + 1))
	}
	wg.Wait()
}
