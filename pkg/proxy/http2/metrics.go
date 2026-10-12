package http2

import (
	"sync/atomic"
	"time"
)

// LatencyTracker uses lock-free atomic buckets for O(1) latency recording.
type LatencyTracker struct {
	buckets [1000]atomic.Uint64
}

func (l *LatencyTracker) Record(d time.Duration) {
	idx := int(d.Microseconds())
	if idx >= 1000 {
		idx = 999
	}
	l.buckets[idx].Add(1)
}

func (l *LatencyTracker) GetPercentile(p float64) time.Duration {
	var total uint64
	for i := 0; i < 1000; i++ {
		total += l.buckets[i].Load()
	}
	if total == 0 {
		return 0
	}
	threshold := uint64(float64(total) * p)
	var count uint64
	for i := 0; i < 1000; i++ {
		count += l.buckets[i].Load()
		if count >= threshold {
			return time.Duration(i) * time.Microsecond
		}
	}
	return 1000 * time.Microsecond
}
