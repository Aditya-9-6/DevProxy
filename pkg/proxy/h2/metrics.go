package h2

import "sync/atomic"

// LatencyBuckets uses fixed-size atomic array for O(1) lock-free recording.
type LatencyBuckets struct {
	buckets [1000]atomic.Uint64
}

func (b *LatencyBuckets) Record(ms uint64) {
	if ms < 1000 {
		b.buckets[ms].Add(1)
	}
}

func (b *LatencyBuckets) GetPercentile(p float64) uint64 {
	var total uint64
	for i := 0; i < 1000; i++ {
		total += b.buckets[i].Load()
	}
	if total == 0 {
		return 0
	}
	var count uint64
	target := uint64(float64(total) * p)
	for i := 0; i < 1000; i++ {
		count += b.buckets[i].Load()
		if count >= target {
			return uint64(i)
		}
	}
	return 1000
}
