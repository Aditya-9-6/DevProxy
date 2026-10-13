package waf

import (
	"math"
	"math/bits"
	"sync/atomic"
)

// HLL represents a HyperLogLog cardinality estimator using atomic registers.
// It is designed for zero-allocation tracking of unique client IPs.
type HLL struct {
	registers []atomic.Uint64
	m         uint32
	p         uint32
}

// NewHLL creates a new HyperLogLog estimator with 2^p registers.
func NewHLL(p uint32) *HLL {
	m := uint32(1 << p)
	regs := make([]atomic.Uint64, m)
	return &HLL{registers: regs, m: m, p: p}
}

// Add inserts a hash value into the HLL structure.
func (h *HLL) Add(hash uint64) {
	idx := hash >> (64 - h.p)
	w := hash<<h.p | (1 << (h.p - 1)) // Ensure at least one bit set
	r := uint64(bits.LeadingZeros64(w) + 1)

	for {
		old := h.registers[idx].Load()
		if old >= r {
			break
		}
		if h.registers[idx].CompareAndSwap(old, r) {
			break
		}
	}
}

// Count returns the estimated cardinality.
func (h *HLL) Count() uint64 {
	var sum float64
	for i := uint32(0); i < h.m; i++ {
		sum += 1.0 / math.Pow(2.0, float64(h.registers[i].Load()))
	}

	alpha := 0.7213 / (1.0 + 1.079/float64(h.m))
	estimate := alpha * float64(h.m*h.m) / sum

	if estimate <= 2.5*float64(h.m) {
		v := 0
		for i := uint32(0); i < h.m; i++ {
			if h.registers[i].Load() == 0 {
				v++
			}
		}
		if v > 0 {
			estimate = float64(h.m) * math.Log(float64(h.m)/float64(v))
		}
	}
	return uint64(estimate)
}
