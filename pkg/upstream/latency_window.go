package upstream

import (
	"sync/atomic"
	"time"
)

const BucketCount = 1024

// LatencyWindow maintains a sliding window of latency observations using atomic buckets.
// Padded to prevent false sharing, and caches stats updated via background sampling or explicit recalculation.
type LatencyWindow struct {
	_           [56]byte // Padding
	buckets     [BucketCount]atomic.Uint64
	cursor      atomic.Uint64
	cachedAvg   atomic.Uint64
	cachedCount atomic.Uint64
	_           [56]byte // Padding
}

// NewLatencyWindow creates and initializes a LatencyWindow, starting a background stats updater.
func NewLatencyWindow() *LatencyWindow {
	lw := &LatencyWindow{}
	// Initialize background stats refresh
	go lw.backgroundStatsLoop()
	return lw
}

func (lw *LatencyWindow) backgroundStatsLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		lw.RefreshStats()
	}
}

// RefreshStats recalculates the average and count across all buckets.
func (lw *LatencyWindow) RefreshStats() (avg uint64, count uint64) {
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
		lw.cachedAvg.Store(0)
		lw.cachedCount.Store(0)
		return 0, 0
	}
	avg = sum / n
	count = n
	lw.cachedAvg.Store(avg)
	lw.cachedCount.Store(count)
	return avg, count
}

// Record adds a latency observation in milliseconds.
func (lw *LatencyWindow) Record(latencyMs uint64) {
	idx := lw.cursor.Add(1) % BucketCount
	lw.buckets[idx].Store(latencyMs)
}

// GetStats returns the cached average and count without performing an O(N) iteration on the hot path.
func (lw *LatencyWindow) GetStats() (avg uint64, count uint64) {
	return lw.cachedAvg.Load(), lw.cachedCount.Load()
}
