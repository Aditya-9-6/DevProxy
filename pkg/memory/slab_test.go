package memory

import (
	"testing"
)

func TestSlabAllocator(t *testing.T) {
	slab := NewSlabAllocator()

	buf := slab.Alloc(100)
	if len(buf) != 128 {
		t.Errorf("expected size 128, got %d", len(buf))
	}

	slab.Free(buf)

	buf2 := slab.Alloc(100)
	if &buf[0] != &buf2[0] {
		t.Error("expected buffer reuse")
	}
}

func TestSlabAllocatorConcurrency(t *testing.T) {
	slab := NewSlabAllocator()
	const workers = 10
	const iterations = 1000
	done := make(chan bool)

	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				buf := slab.Alloc(256)
				slab.Free(buf)
			}
			done <- true
		}()
	}

	for i := 0; i < workers; i++ {
		<-done
	}
}
