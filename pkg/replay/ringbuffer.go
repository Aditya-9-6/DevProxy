package replay

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// TrafficEvent represents a recorded traffic item.
type TrafficEvent struct {
	Seq     uint64
	Payload []byte
}

// RingBufferSlot tracks state and sequence for race-free wrapping writes.
type RingBufferSlot struct {
	seq   atomic.Uint64
	event interface{}
}

// RingBuffer provides a lock-free circular buffer for traffic events with slot-state tracking.
type RingBuffer struct {
	capacity uint64
	mask     uint64
	buffer   []RingBufferSlot
	cursor   atomic.Uint64
	pool     *sync.Pool
}

// NewRingBuffer creates a new lock-free ring buffer of the given size (rounded up to power of 2).
func NewRingBuffer(size uint64) *RingBuffer {
	capacity := uint64(1)
	for capacity < size {
		capacity <<= 1
	}
	buf := make([]RingBufferSlot, capacity)
	for i := range buf {
		buf[i].seq.Store(uint64(i))
	}
	return &RingBuffer{
		capacity: capacity,
		mask:     capacity - 1,
		buffer:   buf,
		pool: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, 32*1024)
				return &b
			},
		},
	}
}

// Push adds an event to the buffer. Returns false if the buffer is full.
func (r *RingBuffer) Push(event interface{}) bool {
	for {
		currentCursor := r.cursor.Load()
		idx := currentCursor & r.mask
		slot := &r.buffer[idx]
		seq := slot.seq.Load()

		diff := int64(seq) - int64(currentCursor)
		if diff == 0 {
			if r.cursor.CompareAndSwap(currentCursor, currentCursor+1) {
				slot.event = event
				slot.seq.Store(currentCursor + 1)
				return true
			}
		} else if diff < 0 {
			// Buffer is full (writer lapped reader)
			return false
		}
		// diff > 0 means another thread is currently writing to this slot
		runtime.Gosched()
	}
}

// GetPool returns the byte slice pool for zero-allocation processing.
func (r *RingBuffer) GetPool() *sync.Pool {
	return r.pool
}

// GetBufferForTest allows unit tests to inspect the buffer state.
func (r *RingBuffer) GetBufferForTest(idx uint64) interface{} {
	return r.buffer[idx&r.mask].event
}
