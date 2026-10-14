package replay

import (
	"sync"
)

// BufferPool manages a sync.Pool of byte slices to eliminate heap allocations
// on hot request/response capture and replay paths.
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool initializes a BufferPool with a specific chunk size.
func NewBufferPool(bufferSize int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, bufferSize)
			},
		},
	}
}

// Get acquires a byte slice from the pool.
func (p *BufferPool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put returns a byte slice to the pool.
func (p *BufferPool) Put(buf []byte) {
	if buf != nil {
		p.pool.Put(buf)
	}
}
