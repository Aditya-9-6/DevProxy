package metrics

import (
	"sync/atomic"
	"time"
)

const (
	MaxLatencyMs = 1000
)

// LatencyTracker provides lock-free latency recording using atomic buckets.
type LatencyTracker struct {
	buckets [MaxLatencyMs]atomic.Uint64
}

func NewLatencyTracker() *LatencyTracker {
	return &LatencyTracker{}
}

// Record adds a latency measurement to the appropriate bucket.
func (l *LatencyTracker) Record(d time.Duration) {
	ms := int(d.Milliseconds())
	if ms >= MaxLatencyMs {
		ms = MaxLatencyMs - 1
	}
	if ms < 0 {
		ms = 0
	}
	l.buckets[ms].Add(1)
}
