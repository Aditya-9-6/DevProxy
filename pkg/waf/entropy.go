package waf

import (
	"github.com/Aditya-9-6/DevProxy/pkg/common"
	"math"
	"sync"
	"sync/atomic"
)

// EntropyScanner computes Shannon entropy using a thread-safe histogram.
type EntropyScanner struct {
	_         common.CacheLinePad
	mu        sync.RWMutex
	histogram [256]atomic.Uint64
	_         common.CacheLinePad
}

// Reset clears the histogram for a new scan.
func (s *EntropyScanner) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < 256; i++ {
		s.histogram[i].Store(0)
	}
}

// Add records byte frequencies.
func (s *EntropyScanner) Add(data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, b := range data {
		s.histogram[b].Add(1)
	}
}

// Calculate returns the Shannon entropy of the recorded data.
func (s *EntropyScanner) Calculate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var counts [256]uint64
	var total uint64
	for i := 0; i < 256; i++ {
		counts[i] = s.histogram[i].Load()
		total += counts[i]
	}
	if total == 0 {
		return 0
	}
	var entropy float64
	for i := 0; i < 256; i++ {
		if counts[i] > 0 {
			p := float64(counts[i]) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}
