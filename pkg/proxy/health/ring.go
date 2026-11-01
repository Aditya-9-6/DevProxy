package health

import (
	"sync"
	"time"
)

// HealthStatus represents the current state of an upstream target.
type HealthStatus int

const (
	StatusUnknown HealthStatus = iota
	StatusHealthy
	StatusUnhealthy
)

func (s HealthStatus) String() string {
	switch s {
	case StatusHealthy:
		return "HEALTHY"
	case StatusUnhealthy:
		return "UNHEALTHY"
	default:
		return "UNKNOWN"
	}
}

// CheckResult holds the result of an individual health check execution.
type CheckResult struct {
	Timestamp time.Time
	Duration  time.Duration
	Status    HealthStatus
	Err       error
}

// RingBuffer implements a fixed-size circular buffer for tracking health check results
// without allocating on the heap during normal operations.
type RingBuffer struct {
	mu       sync.RWMutex
	results  []CheckResult
	head     int
	tail     int
	size     int
	capacity int
}

// NewRingBuffer creates a new ring buffer with the specified capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 10
	}
	return &RingBuffer{
		results:  make([]CheckResult, capacity),
		capacity: capacity,
	}
	}

// Push adds a new check result to the ring buffer, overwriting the oldest if full.
func (rb *RingBuffer) Push(result CheckResult) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.results[rb.tail] = result
	rb.tail = (rb.tail + 1) % rb.capacity

	if rb.size < rb.capacity {
		rb.size++
	} else {
		// Buffer is full, head advances
		rb.head = (rb.head + 1) % rb.capacity
	}
}

// Snapshot returns a slice containing a copy of all current items in chronological order.
func (rb *RingBuffer) Snapshot() []CheckResult {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return nil
	}

	out := make([]CheckResult, rb.size)
	idx := rb.head
	for i := 0; i < rb.size; i++ {
		out[i] = rb.results[idx]
		idx = (idx + 1) % rb.capacity
	}
	return out
}

// CurrentStatus derives the aggregate status based on consecutive failures or successes.
// E.g., if the latest entry is unhealthy, return StatusUnhealthy.
func (rb *RingBuffer) CurrentStatus() HealthStatus {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return StatusUnknown
	}

	// Most recent item is at (tail - 1 + capacity) % capacity
	latestIdx := (rb.tail - 1 + rb.capacity) % rb.capacity
	return rb.results[latestIdx].Status
}
