package congestion

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to 64 bytes.
type CacheLinePad [8]uint64

// ECNTracker tracks ECN-CE (Congestion Experienced) events per upstream peer.
type ECNTracker struct {
	// CongestionCount tracks the number of ECN-CE marked packets received.
	CongestionCount CacheLinePad
}

// NewECNTracker initializes a new tracker.
func NewECNTracker() *ECNTracker {
	return &ECNTracker{}
}

// RecordCongestion increments the congestion event counter atomically.
func (t *ECNTracker) RecordCongestion() {
	atomic.AddUint64(&t.CongestionCount[0], 1)
}

// GetCongestionScore returns the current congestion count.
func (t *ECNTracker) GetCongestionScore() uint64 {
	return atomic.LoadUint64(&t.CongestionCount[0])
}

// DecayCongestion reduces the congestion score over time (e.g., during periodic maintenance).
func (t *ECNTracker) DecayCongestion(factor uint64) {
	for {
		old := atomic.LoadUint64(&t.CongestionCount[0])
		if old == 0 {
			break
		}
		newVal := old / factor
		if atomic.CompareAndSwapUint64(&t.CongestionCount[0], old, newVal) {
			break
		}
	}
}
