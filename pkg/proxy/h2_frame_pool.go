package proxy

import (
	"sync"
)

// FramePool manages slab-allocated buffers for HTTP/2 frame reassembly.
type FramePool struct {
	pools map[int]*sync.Pool
}

// NewFramePool initializes pools for common HTTP/2 frame sizes.
func NewFramePool() *FramePool {
	return &FramePool{
		pools: map[int]*sync.Pool{
			16384: {New: func() interface{} { b := make([]byte, 16384); return b }},
			32768: {New: func() interface{} { b := make([]byte, 32768); return b }},
			65536: {New: func() interface{} { b := make([]byte, 65536); return b }},
		},
	}
}

// Get retrieves a buffer of at least size 'n'.
func (p *FramePool) Get(n int) []byte {
	var size int
	switch {
	case n <= 16384:
		size = 16384
	case n <= 32768:
		size = 32768
	case n <= 65536:
		size = 65536
	default:
		return make([]byte, n)
	}
	return p.pools[size].Get().([]byte)
}

// Put returns a buffer to the appropriate pool.
func (p *FramePool) Put(b []byte) {
	size := cap(b)
	if pool, ok := p.pools[size]; ok {
		pool.Put(b[:cap(b)])
	}
}
