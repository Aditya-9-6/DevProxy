package arena

import "sync"

// SlabAllocator provides a pooled memory management system for byte slices.
// It reduces GC pressure by reusing buffers of fixed sizes.
type SlabAllocator struct {
	pool sync.Pool
	size int
}

func NewSlabAllocator(size int) *SlabAllocator {
	return &SlabAllocator{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

func (a *SlabAllocator) Get() []byte {
	return a.pool.Get().([]byte)
}

func (a *SlabAllocator) Put(b []byte) {
	if len(b) == a.size {
		a.pool.Put(b)
	}
}
