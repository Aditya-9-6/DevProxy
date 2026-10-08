package proxy

import (
	"sync/atomic"
)

// LatencyTracker implements a lock-free HDR-style histogram for latency tracking.
// Uses 1001 atomic buckets representing 1ms increments up to 1000ms, with an overflow bucket.
type LatencyTracker struct {
	buckets [1001]atomic.Uint64
}

// Record adds a latency duration in milliseconds to the histogram.
func (l *LatencyTracker) Record(ms int64) {
	if ms < 0 {
		ms = 0
	}
	if ms > 1000 {
		ms = 1000
	}
	l.buckets[ms].Add(1)
}

// GetPercentiles returns P50, P90, P99, P99.9 in milliseconds with zero heap allocations.
func (l *LatencyTracker) GetPercentiles() (p50, p90, p99, p999 float64) {
	var total uint64
	var snapshot [1001]uint64
	for i := 0; i < 1001; i++ {
		snapshot[i] = l.buckets[i].Load()
		total += snapshot[i]
	}
	if total == 0 {
		return 0, 0, 0, 0
	}

	var count uint64
	var set50, set90, set99, set999 bool
	for i := 0; i < 1001; i++ {
		count += snapshot[i]
		percent := float64(count) / float64(total)
		if !set50 && percent >= 0.50 {
			p50 = float64(i)
			set50 = true
		}
		if !set90 && percent >= 0.90 {
			p90 = float64(i)
			set90 = true
		}
		if !set99 && percent >= 0.99 {
			p99 = float64(i)
			set99 = true
		}
		if !set999 && percent >= 0.999 {
			p999 = float64(i)
			set999 = true
		}
	}
	return
}
