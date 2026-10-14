package health

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing.
type CacheLinePad [8]uint64

// StateRing maintains atomic health status for a node.
type StateRing struct {
	// 0: Healthy, 1: Unhealthy
	Status CacheLinePad
	// Latency buckets for O(1) reporting
	LatencyBuckets [1000]atomic.Uint64
}

func (r *StateRing) SetHealthy(healthy bool) {
	if healthy {
		atomic.StoreUint64(&r.Status[0], 1)
	} else {
		atomic.StoreUint64(&r.Status[0], 0)
	}
}

func (r *StateRing) IsHealthy() bool {
	return atomic.LoadUint64(&r.Status[0]) == 1
}

func (r *StateRing) RecordLatency(ms uint64) {
	if ms >= 1000 {
		ms = 999
	}
	r.LatencyBuckets[ms].Add(1)
}
