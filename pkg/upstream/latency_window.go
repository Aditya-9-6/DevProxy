package upstream

import (
	"sync"
	"sync/atomic"
)

const BucketCount = 1024

// LatencyWindow maintains a sliding window of latency observations using atomic buckets.
// Padded to prevent false sharing.
type LatencyWindow struct {
	_       [56]byte // Padding
	buckets [BucketCount]atomic.Uint64
	cursor  atomic.Uint64
	mu      sync.RWMutex
	_       [56]byte // Padding
}

// Record adds a latency observation in milliseconds.
func (lw *LatencyWindow) Record(latencyMs uint64) {
	idx := lw.cursor.Add(1) % BucketCount
	lw.buckets[idx].Store(latencyMs)
}

// GetStats calculates the average latency using a consistent snapshot.
func (lw *LatencyWindow) GetStats() (avg uint64, count uint64) {
	var sum uint64
	var n uint64
	for i := 0; i < BucketCount; i++ {
		val := lw.buckets[i].Load()
		if val > 0 {
			sum += val
			n++
		}
	}
	if n == 0 {
		return 0, 0
	}
	return sum / n, n
}
