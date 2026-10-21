package health

import (
	"math/bits"
	"sync/atomic"
)

// CacheLinePad is used to pad structs to avoid false sharing.
type CacheLinePad [8]uint64

const healthRingSize = 16

// HealthRing is a lock-free ring buffer for health check results.
// It uses a fixed-size array of cache-line padded slots and a bitmask for fast success counting.
type HealthRing struct {
	_     CacheLinePad
	slots [healthRingSize]healthSlot
	index atomic.Uint64
	bits  atomic.Uint32
	_     CacheLinePad
}

type healthSlot struct {
	latency atomic.Uint64 // in nanoseconds
	_       CacheLinePad
}

// NewHealthRing creates a new HealthRing.
func NewHealthRing() *HealthRing {
	return &HealthRing{}
}

// Record adds a new health check result to the ring.
// success indicates if the check succeeded.
// latency is the check latency in nanoseconds.
func (r *HealthRing) Record(success bool, latency uint64) {
	idx := int(r.index.Add(1) % healthRingSize)
	r.slots[idx].latency.Store(latency)

	// Update bitmask
	mask := r.bits.Load()
	if success {
		mask |= (1 << idx)
	} else {
		mask &^= (1 << idx)
	}
	r.bits.Store(mask)
}

// SuccessRate returns the fraction of successful checks in the ring (0.0 to 1.0).
func (r *HealthRing) SuccessRate() float64 {
	mask := r.bits.Load()
	successes := bits.OnesCount32(mask)
	return float64(successes) / float64(healthRingSize)
}

// AvgLatency returns the average latency of all recorded checks in nanoseconds.
func (r *HealthRing) AvgLatency() float64 {
	var total uint64
	for i := 0; i < healthRingSize; i++ {
		total += r.slots[i].latency.Load()
	}
	return float64(total) / float64(healthRingSize)
}

// Latencies returns a copy of all latencies in the ring.
func (r *HealthRing) Latencies() []uint64 {
	out := make([]uint64, healthRingSize)
	for i := 0; i < healthRingSize; i++ {
		out[i] = r.slots[i].latency.Load()
	}
	return out
}
