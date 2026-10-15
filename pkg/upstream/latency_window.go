package upstream

import (
	"context"
	"sync"
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
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewLatencyWindow creates and initializes a LatencyWindow, starting a background stats updater.
func NewLatencyWindow(ctx context.Context, interval time.Duration) *LatencyWindow {
	ctx, cancel := context.WithCancel(ctx)
	lw := &LatencyWindow{
		cancel: cancel,
	}

	if interval <= 0 {
		interval = 50 * time.Millisecond
	}

	lw.wg.Add(1)
	go lw.backgroundStatsLoop(ctx, interval)
	return lw
}

// Close terminates the background stats loop and waits for it to exit.
func (lw *LatencyWindow) Close() {
	if lw.cancel != nil {
		lw.cancel()
	}
	lw.wg.Wait()
}

func (lw *LatencyWindow) backgroundStatsLoop(ctx context.Context, interval time.Duration) {
	defer lw.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lw.RefreshStats()
		}
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
