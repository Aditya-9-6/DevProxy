package lb

import (
	"testing"
)

func TestConsistentHashing(t *testing.T) {
	nodes := []*Node{
		{ID: "n1", Addr: "127.0.0.1:8001", Capacity: 100},
		{ID: "n2", Addr: "127.0.0.1:8002", Capacity: 100},
	}
	ring := NewRing(nodes, 10)

	// Test basic distribution
	counts := make(map[string]int)
	for i := 0; i < 1000; i++ {
		n := ring.Get("key-" + string(rune(i)))
		counts[n.ID]++
	}

	if counts["n1"] == 0 || counts["n2"] == 0 {
		t.Errorf("Distribution failed: %v", counts)
	}
}

func TestBoundedLoad(t *testing.T) {
	nodes := []*Node{
		{ID: "n1", Addr: "127.0.0.1:8001", Capacity: 1},
	}
	ring := NewRing(nodes, 1)
	nodes[0].Load.Store(2) // Exceed capacity

	n := ring.Get("test")
	if n == nil {
		t.Fatal("Expected node fallback")
	}
}
