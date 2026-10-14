package cache

import (
	"sync"
	"testing"
)

func TestLFUEvictor_Push(t *testing.T) {
	evictor := NewLFUEvictor(10)
	evictor.Push("test", "data")

	if evictor.size != 1 {
		t.Errorf("expected size 1, got %d", evictor.size)
	}
}

func TestLFUEvictor_Eviction(t *testing.T) {
	evictor := NewLFUEvictor(1)
	evictor.Push("k1", "v1")
	evictor.Push("k2", "v2")
	evictor.Evict()

	if evictor.size > 1 {
		t.Errorf("expected size <= 1, got %d", evictor.size)
	}
}

func TestLFUEvictor_Concurrent(t *testing.T) {
	evictor := NewLFUEvictor(100)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			evictor.Push("k", "v")
			evictor.Evict()
		}()
	}
	wg.Wait()
}
