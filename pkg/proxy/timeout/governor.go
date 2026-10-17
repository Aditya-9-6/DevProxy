package timeout

import (
	"sync/atomic"
	"time"
)

// CacheLinePaddedAtomic wraps an atomic.Uint64 with padding to prevent false sharing
// across CPU cache lines (typically 64 or 128 bytes on modern multi-core architectures).
type CacheLinePaddedAtomic struct {
	_   [8]uint64 // Padding bytes / words
	val atomic.Uint64
	_   [8]uint64 // Padding bytes / words
}

// Governor represents a lock-free, cache-line padded adaptive timeout governor
// that tracks upstream response latencies and computes dynamic read/write deadlines
// using an Exponential Moving Average (EMA) variance estimator in O(1) time.
type Governor struct {
	sumLatNs CacheLinePaddedAtomic
	count    CacheLinePaddedAtomic
	emaLatNs CacheLinePaddedAtomic
	varNs    CacheLinePaddedAtomic

	// Configuration defaults
	minTimeout time.Duration
	maxTimeout time.Duration
	alpha      uint64 // Fixed-point EMA alpha (scaled by 1000, e.g., 200 = 0.2)
}

// NewGovernor initializes an adaptive timeout governor with given bounds and alpha smoothing.
func NewGovernor(minTimeout, maxTimeout time.Duration, alphaFloat float64) *Governor {
	if minTimeout <= 0 {
		minTimeout = 500 * time.Millisecond
	}
	if maxTimeout <= minTimeout {
		maxTimeout = 30 * time.Second
	}
	if alphaFloat <= 0 || alphaFloat > 1 {
		alphaFloat = 0.2
	}

	return &Governor{
		minTimeout: minTimeout,
		maxTimeout: maxTimeout,
		alpha:      uint64(alphaFloat * 1000.0),
	}
}

// RecordLatency records an upstream request duration in O(1) time using atomic instructions
// and updates the Exponential Moving Average (EMA) and variance predictor without locks.
func (g *Governor) RecordLatency(d time.Duration) {
	if d <= 0 {
		return
	}

	latNs := uint64(d.Nanoseconds())
	g.sumLatNs.val.Add(latNs)
	cnt := g.count.val.Add(1)

	// Update EMA and variance estimator using lock-free fixed-point math
	alphaVal := g.alpha
	for {
		oldEMA := g.emaLatNs.val.Load()
		var newEMA uint64
		if oldEMA == 0 {
			newEMA = latNs
		} else {
			// EMA = alpha * latNs + (1 - alpha) * oldEMA
			// Scaled by 1000: newEMA = (alpha * latNs + (1000 - alpha) * oldEMA) / 1000
			newEMA = (alphaVal*latNs + (1000-alphaVal)*oldEMA) / 1000
		}

		if g.emaLatNs.val.CompareAndSwap(oldEMA, newEMA) {
			// Calculate absolute deviation / variance component
			var diff uint64
			if latNs > newEMA {
				diff = latNs - newEMA
			} else {
				diff = newEMA - latNs
			}

			for {
				oldVar := g.varNs.val.Load()
				newVar := (alphaVal*diff + (1000-alphaVal)*oldVar) / 1000
				if g.varNs.val.CompareAndSwap(oldVar, newVar) {
					break
				}
			}
			break
		}
	}
	_ = cnt
}

// GetDynamicTimeout computes the adaptive deadline in O(1) time based on EMA + 4 * Standard Deviation.
// Enforces strict zero heap allocations.
func (g *Governor) GetDynamicTimeout() time.Duration {
	ema := g.emaLatNs.val.Load()
	if ema == 0 {
		return g.minTimeout
	}

	deviation := g.varNs.val.Load()
	// Recommended timeout = EMA + 4 * Deviation (approx 99.9th percentile confidence interval)
	totalNs := ema + (4 * deviation)

	dur := time.Duration(totalNs)
	if dur < g.minTimeout {
		return g.minTimeout
	}
	if dur > g.maxTimeout {
		return g.maxTimeout
	}
	return dur
}

// Stats returns current governor metrics: request count, average latency, EMA latency, and variance.
func (g *Governor) Stats() (count uint64, avgLatency, emaLatency, currentVariance time.Duration) {
	c := g.count.val.Load()
	sum := g.sumLatNs.val.Load()
	ema := g.emaLatNs.val.Load()
	variant := g.varNs.val.Load()

	var avg time.Duration
	if c > 0 {
		avg = time.Duration(sum / c)
	}

	return c, avg, time.Duration(ema), time.Duration(variant)
}
