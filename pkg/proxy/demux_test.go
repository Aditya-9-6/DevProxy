package proxy

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestDemuxPool_BasicOperation(t *testing.T) {
	pool := NewDemuxPool(100)

	if pool.ActiveCount() != 0 {
		t.Fatalf("Expected 0 active count initially, got %d", pool.ActiveCount())
	}

	desc := pool.Acquire(42, 12345)
	if desc == nil {
		t.Fatal("Expected non-nil descriptor")
	}

	if desc.GetState() != 42 {
		t.Errorf("Expected state 42, got %d", desc.GetState())
	}

	if !desc.IsActive() {
		t.Errorf("Expected descriptor to be active")
	}

	if pool.ActiveCount() != 1 {
		t.Errorf("Expected 1 active count, got %d", pool.ActiveCount())
	}

	desc.AddBytes(512)
	if desc.GetBytes() != 512 {
		t.Errorf("Expected 512 bytes, got %d", desc.GetBytes())
	}

	pool.Release(desc)
	if pool.ActiveCount() != 0 {
		t.Errorf("Expected 0 active count after release, got %d", pool.ActiveCount())
	}
}

func TestDemuxPool_ConcurrentStress(t *testing.T) {
	pool := NewDemuxPool(1000)
	var wg sync.WaitGroup
	workers := 20
	iterations := 500

	var totalAcquired atomic.Uint64

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				stateID := uint64(workerID*iterations + j)
				desc := pool.Acquire(stateID, uint64(j))
				totalAcquired.Add(1)
				desc.AddBytes(100)
				_ = desc.GetState()
				pool.Release(desc)
			}
		}(i)
	}

	wg.Wait()

	if totalAcquired.Load() != uint64(workers*iterations) {
		t.Errorf("Expected %d total acquired, got %d", workers*iterations, totalAcquired.Load())
	}

	if pool.ActiveCount() != 0 {
		t.Errorf("Expected 0 active count at end, got %d", pool.ActiveCount())
	}
}
