package memory

import (
	"math/bits"
	"sync"
)

const (
	// MinSize is the smallest slab allocation size in bytes (2^6).
	MinSize = 64
	// MaxSize is the largest slab allocation size in bytes (2^16).
	MaxSize = 65536
	// NumClasses is the number of power-of-two size classes between 64 and 64KB.
	NumClasses = 11 // 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536

	maxPerClass   = 1024
	preallocCount = 8
)

type slabClass struct {
	mu      sync.Mutex
	buffers [][]byte
}

// SlabAllocator provides pooled byte slice allocations across power-of-two size classes.
type SlabAllocator struct {
	classes [NumClasses]slabClass
}

// NewSlabAllocator initializes a SlabAllocator with pre-warmed buffer arenas for each size class.
func NewSlabAllocator() *SlabAllocator {
	s := &SlabAllocator{}
	for i := 0; i < NumClasses; i++ {
		classSize := 1 << (i + 6)
		s.classes[i].buffers = make([][]byte, preallocCount, maxPerClass)
		for j := 0; j < preallocCount; j++ {
			s.classes[i].buffers[j] = make([]byte, classSize)
		}
	}
	return s
}

// getClass maps a byte size to its corresponding size class index [0, NumClasses-1].
func (s *SlabAllocator) getClass(size int) int {
	if size <= MinSize {
		return 0
	}
	if size >= MaxSize {
		return NumClasses - 1
	}
	class := bits.Len(uint(size-1)) - 6
	if class < 0 {
		return 0
	}
	if class >= NumClasses {
		return NumClasses - 1
	}
	return class
}

// Alloc returns a byte slice of at least the requested size from the appropriate slab class.
func (s *SlabAllocator) Alloc(size int) []byte {
	if size <= 0 {
		size = MinSize
	}
	if size > MaxSize {
		return make([]byte, size)
	}

	classIdx := s.getClass(size)
	class := &s.classes[classIdx]

	class.mu.Lock()
	n := len(class.buffers)
	if n > 0 {
		buf := class.buffers[n-1]
		class.buffers[n-1] = nil
		class.buffers = class.buffers[:n-1]
		class.mu.Unlock()
		return buf
	}
	class.mu.Unlock()

	return make([]byte, 1<<(classIdx+6))
}

// Free returns a byte slice to its corresponding slab class pool for reuse.
func (s *SlabAllocator) Free(buf []byte) {
	if len(buf) == 0 && cap(buf) == 0 {
		return
	}
	size := cap(buf)
	if size < len(buf) {
		size = len(buf)
	}
	if size < MinSize || size > MaxSize {
		return
	}

	classIdx := s.getClass(size)
	targetSize := 1 << (classIdx + 6)
	if cap(buf) < targetSize {
		return
	}
	buf = buf[:targetSize]

	class := &s.classes[classIdx]
	class.mu.Lock()
	if len(class.buffers) < maxPerClass {
		class.buffers = append(class.buffers, buf)
	}
	class.mu.Unlock()
}
