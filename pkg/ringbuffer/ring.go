package ringbuffer

import (
	"sync"
)

// RingBuffer is a thread-safe, fixed-size circular buffer for traffic events.
type RingBuffer struct {
	mu      sync.Mutex
	data    []*TrafficEvent
	head    int
	tail    int
	size    int
	count   int
	dropped int64
	total   int64
	closed  bool
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]*TrafficEvent, size),
		size: size,
	}
}

func (rb *RingBuffer) Push(event *TrafficEvent) bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.closed {
		return false
	}
	rb.total++
	if rb.count == rb.size {
		rb.dropped++
		return false
	}
	rb.data[rb.tail] = event
	rb.tail = (rb.tail + 1) % rb.size
	rb.count++
	return true
}

func (rb *RingBuffer) Pop() *TrafficEvent {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.count == 0 {
		return nil
	}
	event := rb.data[rb.head]
	rb.head = (rb.head + 1) % rb.size
	rb.count--
	return event
}

func (rb *RingBuffer) TryPop() *TrafficEvent {
	return rb.Pop()
}

func (rb *RingBuffer) Stats() (queued int, dropped int64, total int64) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.count, rb.dropped, rb.total
}

func (rb *RingBuffer) Close() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.closed = true
}
