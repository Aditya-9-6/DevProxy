package cache

import (
	"fmt"
	"sync"
	"testing"
)

func TestLFUEvictor_Push(t *testing.T) {
	evictor := NewLFUEvictor(10)
	evictor.Push("k1", "v1")
	if evictor.size != 1 {
		t.Errorf("expected size 1, got %d", evictor.size)
	}
}

func TestLFUEvictor_Eviction(t *testing.T) {
	evictor := NewLFUEvictor(1)
	evictor.Push("k1", "v1")
	evictor.Push("k2", "v2")
	if _, ok := evictor.nodes["k1"]; ok {
		t.Error("k1 should have been evicted")
	}
}

func TestLFUEvictor_Frequency(t *testing.T) {
	evictor := NewLFUEvictor(2)
	evictor.Push("k1", "v1")
	evictor.Push("k1", "v1")
	evictor.Push("k2", "v2")
	evictor.Push("k3", "v3")
	if _, ok := evictor.nodes["k1"]; !ok {
		t.Error("k1 should remain due to higher frequency")
	}
	if _, ok := evictor.nodes["k2"]; ok {
		t.Error("k2 should have been evicted")
	}
}

func TestLFUEvictor_ZeroCapacity(t *testing.T) {
	evictor := NewLFUEvictor(0)
	evictor.Push("k1", "v1")
	if evictor.size != 0 {
		t.Errorf("expected size 0 for zero capacity cache, got %d", evictor.size)
	}
}

func TestLFUEvictor_UpdateExistingKey(t *testing.T) {
	evictor := NewLFUEvictor(2)
	evictor.Push("k1", "v1")
	evictor.Push("k1", "v1-updated")
	if node, ok := evictor.nodes["k1"]; !ok || node.Value != "v1-updated" {
		t.Error("expected k1 value to be updated")
	}
}

func TestLFUEvictor_RapidEvictionCycles(t *testing.T) {
	evictor := NewLFUEvictor(3)
	for i := 0; i < 100; i++ {
		evictor.Push(fmt.Sprintf("k%d", i), i)
	}
	if evictor.size > 3 {
		t.Errorf("expected size <= 3, got %d", evictor.size)
	}
}

func TestLFUEvictor_ConcurrentStress(t *testing.T) {
	evictor := NewLFUEvictor(50)
	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("key-%d-%d", workerID, i%10)
				evictor.Push(key, i)
			}
		}(w)
	}
	wg.Wait()

	if evictor.size > 50 {
		t.Errorf("concurrent stress test exceeded capacity: got %d", evictor.size)
	}
}
