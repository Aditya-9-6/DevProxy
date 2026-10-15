package congestion

import (
	"sync/atomic"
)

// CongestionWindow implements a lock-free circular buffer for congestion window history and pacing.
type CongestionWindow struct {
	capacity int
	head     atomic.Uint64
	tail     atomic.Uint64
	buffer   []atomic.Uint64
}

// NewCongestionWindow initializes a congestion window tracker with a pre-allocated circular buffer.
func NewCongestionWindow(capacity int) *CongestionWindow {
	if capacity <= 0 {
		capacity = 256
	}
	return &CongestionWindow{
		capacity: capacity,
		buffer:   make([]atomic.Uint64, capacity),
	}
}

// Push records a window size or pacing metric into the circular buffer in a lock-free manner.
func (w *CongestionWindow) Push(val uint64) {
	for {
		head := w.head.Load()
		tail := w.tail.Load()

		// Check if buffer is full
		if (head+1)%uint64(w.capacity) == tail {
			// Advance tail to drop oldest (ring buffer overwrite)
			if w.tail.CompareAndSwap(tail, (tail+1)%uint64(w.capacity)) {
				continue
			}
		}

		if w.head.CompareAndSwap(head, (head+1)%uint64(w.capacity)) {
			w.buffer[head%uint64(w.capacity)].Store(val)
			break
		}
	}
}

// CurrentWindow returns the latest congestion window estimate.
func (w *CongestionWindow) CurrentWindow() uint64 {
	head := w.head.Load()
	tail := w.tail.Load()
	if head == tail {
		return 0
	}
	prevIndex := (head - 1 + uint64(w.capacity)) % uint64(w.capacity)
	return w.buffer[prevIndex].Load()
}
