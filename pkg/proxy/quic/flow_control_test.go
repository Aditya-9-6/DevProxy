package quic

import (
	"sync"
	"testing"
)

func TestWindowManager_Concurrency(t *testing.T) {
	initial := uint64(10000)
	wm := NewWindowManager(initial)
	workers := 10
	iterations := 1000
	var wg sync.WaitGroup

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if wm.Consume(1) {
					wm.Replenish(1)
				}
			}
		}()
	}
	wg.Wait()

	if wm.Available() != initial {
		t.Errorf("expected window %d, got %d", initial, wm.Available())
	}
}

func TestWindowManager_Boundary(t *testing.T) {
	wm := NewWindowManager(100)
	if !wm.Consume(100) {
		t.Error("failed to consume full window")
	}
	if wm.Consume(1) {
		t.Error("should not be able to consume beyond window")
	}
	wm.Replenish(50)
	if wm.Available() != 50 {
		t.Errorf("expected 50, got %d", wm.Available())
	}
}
