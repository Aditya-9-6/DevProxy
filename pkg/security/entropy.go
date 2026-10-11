package security

import (
	"math"
	"sync/atomic"
)

// EntropyAnalyzer provides high-performance, zero-allocation Shannon entropy calculation.
// It uses atomic counters to track byte frequencies, making it safe for concurrent use.
type EntropyAnalyzer struct {
	frequencies [256]atomic.Uint64
}

// NewEntropyAnalyzer creates a new analyzer instance.
func NewEntropyAnalyzer() *EntropyAnalyzer {
	return &EntropyAnalyzer{}
}

// Reset clears the frequency counters for a new analysis window.
func (e *EntropyAnalyzer) Reset() {
	for i := 0; i < 256; i++ {
		e.frequencies[i].Store(0)
	}
}

// Add processes a chunk of data, updating frequency counts atomically.
func (e *EntropyAnalyzer) Add(data []byte) {
	for _, b := range data {
		e.frequencies[b].Add(1)
	}
}

// Calculate returns the Shannon entropy of the processed data.
// Entropy ranges from 0 (completely predictable) to 8 (random/encrypted).
func (e *EntropyAnalyzer) Calculate() float64 {
	var total uint64
	for i := 0; i < 256; i++ {
		total += e.frequencies[i].Load()
	}

	if total == 0 {
		return 0
	}

	var entropy float64
	invTotal := 1.0 / float64(total)
	for i := 0; i < 256; i++ {
		count := e.frequencies[i].Load()
		if count > 0 {
			p := float64(count) * invTotal
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}
