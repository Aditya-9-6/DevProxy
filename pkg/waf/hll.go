package waf

import (
	"math"
)

// HyperLogLog implements cardinality estimation with low memory footprint.
type HyperLogLog struct {
	registers []byte
	bits      uint
	m         uint
}

// NewHyperLogLog creates a new HLL structure with precision p.
func NewHyperLogLog(p uint) *HyperLogLog {
	if p < 4 || p > 16 {
		p = 12
	}
	m := uint(1) << p
	return &HyperLogLog{
		registers: make([]byte, m),
		bits:      p,
		m:         m,
	}
}

// Add inserts a hash into the HLL.
func (h *HyperLogLog) Add(hash uint64) {
	idx := hash >> (64 - h.bits)
	val := rho(hash<<h.bits | (1<<(h.bits-1)), 64-h.bits)
	if val > h.registers[idx] {
		h.registers[idx] = val
	}
}

func rho(val uint64, maxBits uint) byte {
	if val == 0 {
		return byte(maxBits + 1)
	}
	var r byte = 1
	for (val & (1 << 63)) == 0 {
		r++
		val <<= 1
	}
	return r
}

// Count estimates the number of distinct elements.
func (h *HyperLogLog) Count() uint64 {
	mFloat := float64(h.m)
	alpha := 0.7213 / (1.0 + 1.079/mFloat)
	if h.bits == 16 {
		alpha = 0.7213 / (1.0 + 1.079/mFloat)
	}

	sum := 0.0
	for _, val := range h.registers {
		sum += math.Pow(2.0, -float64(val))
	}

	estimate := alpha * mFloat * mFloat / sum
	return uint64(estimate)
}
