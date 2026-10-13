package tunnel

import (
	"sync"
)

// BufferPool manages zero-allocation byte slice allocations via sync.Pool.
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a buffer pool with specified default buffer size.
func NewBufferPool(size int) *BufferPool {
	if size <= 0 {
		size = 4096
	}
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get acquires a byte slice from the pool.
func (bp *BufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

// Put returns a byte slice to the pool after clearing or resetting if needed.
func (bp *BufferPool) Put(b []byte) {
	if b != nil {
		bp.pool.Put(b)
	}
}
