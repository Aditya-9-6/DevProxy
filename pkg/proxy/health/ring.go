package health

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding to 64 bytes.
// The struct is aligned to 64 bytes to fit a standard CPU cache line.
type CacheLinePad struct {
	V atomic.Uint64
	_ [56]byte
}

// StateRing maintains atomic health status for a node.
// It is designed for high-throughput concurrent access.
type StateRing struct {
	Status CacheLinePad
	// LatencyBuckets uses a smaller, more efficient histogram representation
	// to reduce memory footprint while maintaining O(1) access.
	LatencyBuckets [100]atomic.Uint64
}

func (r *StateRing) SetHealthy(healthy bool) {
	if healthy {
		r.Status.V.Store(1)
	} else {
		r.Status.V.Store(0)
	}
}

func (r *StateRing) IsHealthy() bool {
	return r.Status.V.Load() == 1
}

func (r *StateRing) RecordLatency(ms uint64) {
	// Map latency to 100 buckets (0-990ms, 10ms increments)
	idx := ms / 10
	if idx >= 100 {
		idx = 99
	}
	r.LatencyBuckets[idx].Add(1)
}
