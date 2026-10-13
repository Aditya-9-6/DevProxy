package concurrency

import (
	"sync/atomic"
	"time"
)

const (
	MaxLatencyBuckets  = 1000
	BucketResolutionMs = 10
)

type LatencyTracker struct {
	buckets [MaxLatencyBuckets]atomic.Uint64
}

func (l *LatencyTracker) Record(latency time.Duration) {
	idx := int(latency.Milliseconds() / BucketResolutionMs)
	if idx >= MaxLatencyBuckets {
		idx = MaxLatencyBuckets - 1
	}
	l.buckets[idx].Add(1)
}

func (l *LatencyTracker) GetPercentile(p float64) time.Duration {
	var total uint64
	for i := 0; i < MaxLatencyBuckets; i++ {
		total += l.buckets[i].Load()
	}
	if total == 0 {
		return 0
	}
	target := uint64(p * float64(total))
	var count uint64
	for i := 0; i < MaxLatencyBuckets; i++ {
		count += l.buckets[i].Load()
		if count >= target {
			return time.Duration(i*BucketResolutionMs) * time.Millisecond
		}
	}
	return time.Duration(MaxLatencyBuckets*BucketResolutionMs) * time.Millisecond
}
