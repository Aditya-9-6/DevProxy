package cache

import (
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
