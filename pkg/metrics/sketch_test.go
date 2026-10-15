package metrics

import (
	"sync"
	"testing"
)

func TestCountMinSketch_Accuracy(t *testing.T) {
	sketch := NewCountMinSketch()
	key := "upstream-error-500"
	count := 100

	for i := 0; i < count; i++ {
		sketch.Increment(key)
	}

	estimate := sketch.Estimate(key)
	if estimate < uint64(count) {
		t.Errorf("expected estimate >= %d, got %d", count, estimate)
	}
}

func TestCountMinSketch_Concurrency(t *testing.T) {
	sketch := NewCountMinSketch()
	key := "concurrent-key"
	workers := 10
	increments := 1000

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				sketch.Increment(key)
			}
		}()
	}
	wg.Wait()

	estimate := sketch.Estimate(key)
	if estimate != uint64(workers*increments) {
		t.Errorf("expected %d, got %d", workers*increments, estimate)
	}
}
