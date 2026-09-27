package ringbuffer

import (
	"sync"
	"sync/atomic"
)

// RingBuffer is a high-throughput bounded circular ring buffer designed to decouple
// the real-time proxy data path from the asynchronous security analysis path.
type RingBuffer struct {
	buffer     []*TrafficEvent
	capacity   uint64
	mask       uint64
	head       atomic.Uint64 // write pointer
	tail       atomic.Uint64 // read pointer
	dropped    atomic.Uint64 // count of dropped events if analysis lags
	totalPushed atomic.Uint64
	mu         sync.Mutex    // lightweight mutex for multi-consumer synchronization & cond
	cond       *sync.Cond
	closed     atomic.Bool
}

// NewRingBuffer creates a circular ring buffer with the given size (rounded up to power of 2).
func NewRingBuffer(size int) *RingBuffer {
	if size < 16 {
		size = 16
	}
	// Round up to power of 2
	cap := uint64(1)
	for cap < uint64(size) {
		cap <<= 1
	}

	rb := &RingBuffer{
		buffer:   make([]*TrafficEvent, cap),
		capacity: cap,
		mask:     cap - 1,
	}
	rb.cond = sync.NewCond(&rb.mu)
	return rb
}

// Push adds an event to the ring buffer. It is completely non-blocking:
// if the analysis pipeline falls behind and the buffer is full, it drops the item
// and immediately returns to prevent any latency degradation in the proxy data path.
func (rb *RingBuffer) Push(event *TrafficEvent) bool {
	if rb.closed.Load() {
		return false
	}

	rb.mu.Lock()
	head := rb.head.Load()
	tail := rb.tail.Load()

	if head-tail >= rb.capacity {
		// Buffer is full. Never block data path!
		rb.dropped.Add(1)
		rb.mu.Unlock()
		return false
	}

	idx := head & rb.mask
	rb.buffer[idx] = event
	rb.head.Add(1)
	rb.totalPushed.Add(1)

	rb.cond.Signal()
	rb.mu.Unlock()
	return true
}

// Pop retrieves the next event from the buffer. It blocks until an event is available or the buffer is closed.
func (rb *RingBuffer) Pop() *TrafficEvent {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for rb.head.Load() == rb.tail.Load() {
		if rb.closed.Load() {
			return nil
		}
		rb.cond.Wait()
	}

	tail := rb.tail.Load()
	idx := tail & rb.mask
	event := rb.buffer[idx]
	rb.buffer[idx] = nil // allow GC
	rb.tail.Add(1)

	return event
}

// TryPop attempts to retrieve an event without blocking. Returns nil if empty.
func (rb *RingBuffer) TryPop() *TrafficEvent {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.head.Load() == rb.tail.Load() {
		return nil
	}

	tail := rb.tail.Load()
	idx := tail & rb.mask
	event := rb.buffer[idx]
	rb.buffer[idx] = nil
	rb.tail.Add(1)

	return event
}

// Stats returns current operational metrics.
func (rb *RingBuffer) Stats() (queued uint64, dropped uint64, total uint64) {
	head := rb.head.Load()
	tail := rb.tail.Load()
	queued = 0
	if head > tail {
		queued = head - tail
	}
	return queued, rb.dropped.Load(), rb.totalPushed.Load()
}

// Close shuts down the buffer and wakes up waiting consumers.
func (rb *RingBuffer) Close() {
	if rb.closed.Swap(true) {
		return
	}
	rb.mu.Lock()
	rb.cond.Broadcast()
	rb.mu.Unlock()
}
