package waf

import (
	"github.com/Aditya-9-6/DevProxy/pkg/internal/align"
	"sync"
)

// CacheLinePad ensures cache line alignment to prevent false sharing.
type CacheLinePad = align.CacheLinePad

// Arena provides pre-allocated fixed memory buffers to guarantee zero GC overhead on the hot path.
type Arena struct {
	pool       sync.Pool
	bufferSize int
}

// NewArena creates a new memory arena backed by a sync.Pool of pre-allocated buffers.
func NewArena(bufferSize int) *Arena {
	return &Arena{
		bufferSize: bufferSize,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, bufferSize)
			},
		},
	}
}

// Get acquires a byte slice from the arena pool.
func (a *Arena) Get() []byte {
	return a.pool.Get().([]byte)
}

// Put returns a byte slice back to the arena pool if it matches the expected capacity.
func (a *Arena) Put(buf []byte) {
	if cap(buf) >= a.bufferSize {
		a.pool.Put(buf[:0])
	}
}
