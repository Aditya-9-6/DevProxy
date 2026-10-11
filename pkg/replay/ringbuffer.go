package replay

import (
	"sync"
	"sync/atomic"
)

// RingBuffer provides a lock-free circular buffer for traffic events.
type RingBuffer struct {
	capacity uint64
	mask     uint64
	buffer   []interface{}
	cursor   atomic.Uint64
	pool     *sync.Pool
}

func NewRingBuffer(size uint64) *RingBuffer {
	// Ensure power of 2 for bitwise masking
	capacity := uint64(1)
	for capacity < size {
		capacity <<= 1
	}
	return &RingBuffer{
		capacity: capacity,
		mask:     capacity - 1,
		buffer:   make([]interface{}, capacity),
		pool: &sync.Pool{
			New: func() interface{} { return make([]byte, 32*1024) },
		},
	}
}

func (r *RingBuffer) Push(event interface{}) {
	idx := r.cursor.Add(1) - 1
	r.buffer[idx&r.mask] = event
}

func (r *RingBuffer) GetPool() *sync.Pool {
	return r.pool
}
