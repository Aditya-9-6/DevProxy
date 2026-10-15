package upstream

import (
	"sync/atomic"
)

// CacheLinePaddedCounter prevents false sharing across CPU cores by padding atomic counters.
type CacheLinePaddedCounter struct {
	_value atomic.Uint64
	_pad   [7]uint64 // Pad to 64 bytes cache line alignment
}

// AtomicTargetStats tracks active stream counts and throughput metrics using lock-free atomic primitives.
type AtomicTargetStats struct {
	activeStreams CacheLinePaddedCounter
	totalRequests CacheLinePaddedCounter
}

// NewAtomicTargetStats allocates and returns a new AtomicTargetStats instance.
func NewAtomicTargetStats() *AtomicTargetStats {
	return &AtomicTargetStats{}
}

// IncStreams atomically increments the active stream count.
func (s *AtomicTargetStats) IncStreams() {
	s.activeStreams._value.Add(1)
	s.totalRequests._value.Add(1)
}

// DecStreams atomically decrements the active stream count.
func (s *AtomicTargetStats) DecStreams() {
	// Decrement with underflow safety protection
	for {
		current := s.activeStreams._value.Load()
		if current == 0 {
			break
		}
		if s.activeStreams._value.CompareAndSwap(current, current-1) {
			break
		}
	}
}

// GetActiveStreams retrieves the current active stream count using a lock-free atomic load.
func (s *AtomicTargetStats) GetActiveStreams() uint64 {
	return s.activeStreams._value.Load()
}

// GetTotalRequests retrieves total requests routed through this target.
func (s *AtomicTargetStats) GetTotalRequests() uint64 {
	return s.totalRequests._value.Load()
}
