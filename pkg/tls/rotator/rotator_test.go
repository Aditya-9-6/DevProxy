package rotator

import (
	"sync"
	"testing"
)

func TestRotator(t *testing.T) {
	r := NewRotator()
	key1 := r.Get()
	if key1 == nil {
		t.Fatal("expected initial key")
	}

	r.Rotate()
	key2 := r.Get()
	if key2 == nil || *key1 == *key2 {
		t.Fatal("expected new key after rotation")
	}
}

func TestRotatorConcurrent(t *testing.T) {
	t.Parallel()
	r := NewRotator()

	const numGoroutines = 16
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Goroutines repeatedly reading active key
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := r.Get()
				if key == nil {
					t.Errorf("expected non-nil STEK")
					return
				}
			}
		}()
	}

	// Goroutines repeatedly rotating active key
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				r.Rotate()
				key := r.Get()
				if key == nil {
					t.Errorf("expected non-nil STEK after rotation")
					return
				}
			}
		}()
	}

	wg.Wait()
}
