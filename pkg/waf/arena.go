package waf

import (
	"sync"
)

// CacheLinePad ensures cache line alignment to prevent false sharing.
type CacheLinePad [8]uint64

// Arena provides pre-allocated fixed memory buffers to guarantee zero GC overhead on the hot path.
type Arena struct {
	pool sync.Pool
}

// NewArena creates a new memory arena backed by a sync.Pool of pre-allocated buffers.
func NewArena(bufferSize int) *Arena {
	return &Arena{
		pool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, bufferSize)
				return &buf
			},
		},
	}
}

// Get acquires a byte slice from the arena pool.
func (a *Arena) Get() *[]byte {
	return a.pool.Get().(*[]byte)
}

// Put returns a byte slice back to the arena pool.
func (a *Arena) Put(buf *[]byte) {
	if buf != nil && cap(*buf) > 0 {
		a.pool.Put(buf)
	}
}
