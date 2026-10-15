package proxy

import (
	"sync"
)

// DeflateBufferPool provides zero-allocation byte slice recycling for compression streams.
type DeflateBufferPool struct {
	pool sync.Pool
}

// NewDeflateBufferPool initializes a buffer pool with a specified default slice capacity.
func NewDeflateBufferPool(bufferSize int) *DeflateBufferPool {
	if bufferSize <= 0 {
		bufferSize = 32768
	}
	return &DeflateBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				b := make([]byte, 0, bufferSize)
				return &b
			},
		},
	}
}

// Get retrieves a pre-allocated byte slice pointer from the pool.
func (p *DeflateBufferPool) Get() *[]byte {
	return p.pool.Get().(*[]byte)
}

// Put returns a byte slice pointer back to the pool after resetting its length.
func (p *DeflateBufferPool) Put(b *[]byte) {
	if b == nil {
		return
	}
	*b = (*b)[:0]
	p.pool.Put(b)
}

// Global shared deflation buffer pool
var DefaultDeflatePool = NewDeflateBufferPool(32768)
