package memory

import (
	"bytes"
	"sync"
	"testing"
)

func TestSlabAllocator(t *testing.T) {
	slab := NewSlabAllocator()

	buf := slab.Alloc(100)
	if len(buf) != 128 {
		Errorf(t, "expected size 128, got %d", len(buf))
	}

	slab.Free(buf)

	buf2 := slab.Alloc(100)
	if &buf[0] != &buf2[0] {
		t.Error("expected buffer reuse")
	}
}

func Errorf(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	t.Errorf(format, args...)
}

func TestSlabAllocatorEdgeCases(t *testing.T) {
	slab := NewSlabAllocator()

	// Negative or zero size
	zeroBuf := slab.Alloc(0)
	if len(zeroBuf) < MinSize {
		t.Errorf("expected at least MinSize, got %d", len(zeroBuf))
	}
	slab.Free(zeroBuf)

	negBuf := slab.Alloc(-100)
	if len(negBuf) < MinSize {
		t.Errorf("expected at least MinSize for negative size, got %d", len(negBuf))
	}
	slab.Free(negBuf)

	// Oversized allocation
	overSize := slab.Alloc(MaxSize + 1024)
	if len(overSize) != MaxSize+1024 {
		t.Errorf("expected oversized allocation, got %d", len(overSize))
	}
	slab.Free(overSize) // Should safely ignore

	// Nil and empty free
	slab.Free(nil)
	slab.Free([]byte{})

	// Free buffer with capacity below target class
	badBuf := make([]byte, 10, 50)
	slab.Free(badBuf)
}

func TestSlabAllocator_AllClasses(t *testing.T) {
	slab := NewSlabAllocator()

	for i := 0; i < NumClasses; i++ {
		expectedSize := 1 << (i + 6)
		buf := slab.Alloc(expectedSize)
		if len(buf) != expectedSize {
			t.Fatalf("class %d: expected size %d, got %d", i, expectedSize, len(buf))
		}
		if cap(buf) != expectedSize {
			t.Fatalf("class %d: expected cap %d, got %d", i, expectedSize, cap(buf))
		}
		buf[0] = 0xAA
		buf[len(buf)-1] = 0xBB
		slab.Free(buf)

		reused := slab.Alloc(expectedSize)
		if &buf[0] != &reused[0] {
			t.Fatalf("class %d: expected reused buffer", i)
		}
	}
}

func TestSlabAllocator_IntegrityAndCapacity(t *testing.T) {
	slab := NewSlabAllocator()

	// Test boundary sizes around power of two thresholds
	sizes := []int{63, 64, 65, 127, 128, 129, 255, 256, 1000, 4096, 65535, 65536}
	for _, sz := range sizes {
		buf := slab.Alloc(sz)
		if len(buf) < sz {
			t.Fatalf("requested %d but got buffer of len %d", sz, len(buf))
		}

		pattern := make([]byte, len(buf))
		for k := range pattern {
			pattern[k] = byte(k % 251)
		}
		copy(buf, pattern)

		if !bytes.Equal(buf, pattern) {
			t.Fatalf("data corruption detected for size %d", sz)
		}

		slab.Free(buf)
	}
}

func TestSlabAllocator_ExhaustAndRefill(t *testing.T) {
	slab := NewSlabAllocator()

	const allocCount = 50
	allocated := make([][]byte, allocCount)
	for i := 0; i < allocCount; i++ {
		allocated[i] = slab.Alloc(256)
		if len(allocated[i]) != 256 {
			t.Fatalf("expected 256, got %d", len(allocated[i]))
		}
	}

	for i := 0; i < allocCount; i++ {
		slab.Free(allocated[i])
	}

	for i := 0; i < allocCount; i++ {
		buf := slab.Alloc(256)
		if len(buf) != 256 {
			t.Fatalf("expected 256, got %d", len(buf))
		}
	}
}

func TestSlabAllocatorConcurrency(t *testing.T) {
	slab := NewSlabAllocator()
	const workers = 20
	const iterations = 2000
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				size := MinSize << (j % NumClasses)
				if size > MaxSize {
					size = MaxSize
				}
				buf := slab.Alloc(size)
				buf[0] = byte(workerID)
				buf[len(buf)-1] = byte(workerID ^ 0xFF)
				slab.Free(buf)
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkSlabAllocator_AllocFree(b *testing.B) {
	slab := NewSlabAllocator()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := slab.Alloc(512)
		slab.Free(buf)
	}
}
