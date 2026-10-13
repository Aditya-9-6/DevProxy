package telemetry

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to 64 bytes.
type CacheLinePad [8]uint64

// Sampler enforces a dynamic sampling rate using atomic operations.
type Sampler struct {
	// counters[0]: total requests, counters[1]: sampled requests, counters[2]: error count
	counters [3]CacheLinePad
	// rate is stored as a fixed-point integer (0-10000 representing 0.00% to 100.00%)
	rate atomic.Uint64
}

func NewSampler(initialRate uint64) *Sampler {
	s := &Sampler{}
	s.rate.Store(initialRate)
	return s
}

// ShouldSample returns true if the request should be traced, using atomic operations.
func (s *Sampler) ShouldSample() bool {
	atomic.AddUint64(&s.counters[0][0], 1)
	rate := s.rate.Load()
	// Simple atomic pseudo-random sampling
	if (atomic.AddUint64(&s.counters[1][0], 1) % 10000) < rate {
		return true
	}
	return false
}

// RecordError updates the error count and adjusts the sampling rate via EMA feedback.
func (s *Sampler) RecordError() {
	atomic.AddUint64(&s.counters[2][0], 1)
	// EMA feedback: Increase sampling rate if errors are detected
	for {
		oldRate := s.rate.Load()
		if oldRate >= 10000 {
			break
		}
		newRate := oldRate + 100
		if newRate > 10000 {
			newRate = 10000
		}
		if s.rate.CompareAndSwap(oldRate, newRate) {
			break
		}
	}
}

func (s *Sampler) GetStats() (total, sampled, errors uint64) {
	return atomic.LoadUint64(&s.counters[0][0]), atomic.LoadUint64(&s.counters[1][0]), atomic.LoadUint64(&s.counters[2][0])
}
