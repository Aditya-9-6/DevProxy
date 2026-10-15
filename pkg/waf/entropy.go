package waf

import (
	"github.com/Aditya-9-6/DevProxy/pkg/proxy"
	"math"
	"sync/atomic"
)

// EntropyScanner computes Shannon entropy using a lock-free atomic histogram.
type EntropyScanner struct {
	histogram [256]atomic.Uint64
}

// Reset clears the histogram for a new scan.
func (s *EntropyScanner) Reset() {
	for i := 0; i < 256; i++ {
		s.histogram[i].Store(0)
	}
}

// Add records byte frequencies.
func (s *EntropyScanner) Add(data []byte) {
	for _, b := range data {
		s.histogram[b].Add(1)
	}
}

// Calculate returns the Shannon entropy of the recorded data.
func (s *EntropyScanner) Calculate() float64 {
	var total uint64
	for i := 0; i < 256; i++ {
		total += s.histogram[i].Load()
	}
	if total == 0 {
		return 0
	}
	var entropy float64
	for i := 0; i < 256; i++ {
		count := s.histogram[i].Load()
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

var _ = proxy.CacheLinePad{}
