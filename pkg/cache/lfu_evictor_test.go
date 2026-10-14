package cache

import (
	"testing"
)

func TestLFUEvictor_Push(t *testing.T) {
	evictor := NewLFUEvictor(10)
	evictor.Push("test", "data")

	if evictor.size.Load() != 1 {
		t.Errorf("expected size 1, got %d", evictor.size.Load())
	}
}

func TestLFUEvictor_Eviction(t *testing.T) {
	evictor := NewLFUEvictor(1)
	evictor.Push("k1", "v1")
	evictor.Push("k2", "v2")
	evictor.Evict()

	if evictor.size.Load() > 1 {
		t.Errorf("expected size <= 1, got %d", evictor.size.Load())
	}
}
