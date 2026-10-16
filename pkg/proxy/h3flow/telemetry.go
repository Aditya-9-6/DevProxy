package h3flow

import (
	"sync/atomic"
)

const BucketCount = 1000

// TelemetryEngine tracks RTT and congestion metrics using lock-free atomic buckets.
type TelemetryEngine struct {
	buckets [BucketCount]atomic.Uint64
}

func NewTelemetryEngine() *TelemetryEngine {
	return &TelemetryEngine{}
}

// RecordRTT records a round-trip time in milliseconds.
func (t *TelemetryEngine) RecordRTT(rttMs uint64) {
	if rttMs >= BucketCount {
		rttMs = BucketCount - 1
	}
	t.buckets[rttMs].Add(1)
}

// GetPercentile calculates the RTT percentile without heap allocations.
func (t *TelemetryEngine) GetPercentile(percentile float64) uint64 {
	var total uint64
	for i := 0; i < BucketCount; i++ {
		total += t.buckets[i].Load()
	}
	if total == 0 {
		return 0
	}

	target := uint64(float64(total) * percentile)
	var current uint64
	for i := 0; i < BucketCount; i++ {
		current += t.buckets[i].Load()
		if current >= target {
			return uint64(i)
		}
	}
	return BucketCount - 1
}
