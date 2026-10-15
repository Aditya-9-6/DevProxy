package telemetry

import (
	"math"
	"sync/atomic"
)

const (
	NumBuckets = 64
	MinLatency = 1000 // 1us
)

// Histogram tracks latency using exponential buckets.
type Histogram struct {
	buckets [NumBuckets]atomic.Uint64
}

// Record adds a latency observation in nanoseconds.
func (h *Histogram) Record(latencyNs uint64) {
	if latencyNs < MinLatency {
		latencyNs = MinLatency
	}
	idx := int(math.Log2(float64(latencyNs) / MinLatency))
	if idx < 0 {
		idx = 0
	} else if idx >= NumBuckets {
		idx = NumBuckets - 1
	}
	h.buckets[idx].Add(1)
}

// GetPercentile calculates the approximate percentile without heap allocations.
func (h *Histogram) GetPercentile(p float64) uint64 {
	if p <= 0 {
		return 0
	}
	if p >= 1 {
		p = 0.999
	}

	var total uint64
	var counts [NumBuckets]uint64
	for i := 0; i < NumBuckets; i++ {
		counts[i] = h.buckets[i].Load()
		total += counts[i]
	}

	if total == 0 {
		return 0
	}

	target := uint64(float64(total) * p)
	var current uint64
	for i := 0; i < NumBuckets; i++ {
		current += counts[i]
		if current >= target {
			return uint64(MinLatency * math.Pow(2, float64(i)))
		}
	}
	return uint64(MinLatency * math.Pow(2, float64(NumBuckets-1)))
}
