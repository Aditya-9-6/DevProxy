package metrics

// GetPercentiles calculates latency percentiles without heap allocations.
// It iterates over the pre-allocated buckets directly.
func (l *LatencyTracker) GetPercentiles(p50, p90, p99 *float64) {
	var total uint64
	for i := 0; i < MaxLatencyMs; i++ {
		total += l.buckets[i].Load()
	}

	if total == 0 {
		return
	}

	var count uint64
	for i := 0; i < MaxLatencyMs; i++ {
		count += l.buckets[i].Load()
		percent := float64(count) / float64(total)

		if p50 != nil && *p50 == 0 && percent >= 0.50 {
			*p50 = float64(i)
		}
		if p90 != nil && *p90 == 0 && percent >= 0.90 {
			*p90 = float64(i)
		}
		if p99 != nil && *p99 == 0 && percent >= 0.99 {
			*p99 = float64(i)
		}
	}
}
