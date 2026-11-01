package metrics

import (
	"sync"
	"testing"
)

func TestSketchBasic(t *testing.T) {
	sketch := NewSketch(4, 512)

	key := []byte("/api/v1/upstream-error")
	sketch.Add(key, 1)
	sketch.Add(key, 5)

	val := sketch.Estimate(key)
	if val < 6 {
		t.Errorf("expected estimate to be at least 6, got %d", val)
	}
}

func TestSketchMultipleKeys(t *testing.T) {
	sketch := NewSketch(3, 256)

	keyA := []byte("GET /healthz")
	keyB := []byte("POST /submit")

	sketch.Add(keyA, 10)
	sketch.Add(keyB, 20)

	if estA := sketch.Estimate(keyA); estA < 10 {
		t.Errorf("keyA estimate incorrect: %d", estA)
	}
	if estB := sketch.Estimate(keyB); estB < 20 {
		t.Errorf("keyB estimate incorrect: %d", estB)
	}
}

func TestSketchReset(t *testing.T) {
	sketch := NewSketch(2, 128)
	key := []byte("test-reset")

	sketch.Add(key, 50)
	if sketch.Estimate(key) < 50 {
		t.Errorf("expected non-zero count before reset")
	}

	sketch.Reset()
	if val := sketch.Estimate(key); val != 0 {
		t.Errorf("expected 0 after reset, got %d", val)
	}
}

func TestSketchConcurrentSafety(t *testing.T) {
	sketch := NewSketch(5, 1024)
	key := []byte("/concurrent/path")

	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				sketch.Add(key, 1)
			}
		}()
	}

	wg.Wait()

	expected := uint64(workers * iterations)
	val := sketch.Estimate(key)
	if val < expected {
		t.Errorf("expected estimate at least %d under concurrency, got %d", expected, val)
	}
}
