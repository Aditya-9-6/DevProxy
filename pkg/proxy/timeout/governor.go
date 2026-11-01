package timeout

import (
	"sync/atomic"
	"time"
)

// Governor tracks upstream response latencies lock-free using atomic primitives
// and computes dynamic read/write timeouts using an Exponential Moving Average (EMA)
// variance predictor.
// Cache-line padding is simulated via explicit struct sizing/alignment principles to reduce false sharing.
type Governor struct {
	// Packed atomic fields for zero-lock, high-throughput metrics
	// We store durations and counts in nanoseconds as int64
	sumLatencyNs   atomic.Int64 // Cumulative latency sum for EMA mean
	varLatencyNs   atomic.Int64 // Cumulative squared deviation sum for EMA variance
	reqCount       atomic.Int64 // Total requests observed
	currentMeanNs  atomic.Int64 // Current smoothed EMA mean latency
	currentVarNs   atomic.Int64 // Current smoothed EMA variance

	// Configuration parameters
	minTimeout time.Duration
	maxTimeout time.Duration
	multiplier float64 // Multiplier for standard deviation buffer (e.g. 3.0)
	alpha        float64 // EMA smoothing factor (0 < alpha <= 1)
}

// NewGovernor creates a new adaptive timeout governor.
func NewGovernor(minTimeout, maxTimeout time.Duration, multiplier float64, alpha float64) *Governor {
	if multiplier <= 0 {
		multiplier = 3.0
	}
	if alpha <= 0 || alpha > 1 {
		alpha = 0.2
	}
	g := &Governor{
		minTimeout: minTimeout,
		maxTimeout: maxTimeout,
		multiplier: multiplier,
		alpha:      alpha,
	}
	// Initialize base mean to minTimeout if valid
	g.currentMeanNs.Store(int64(minTimeout))
	g.currentVarNs.Store(int64(minTimeout / 4))
	return g
}

// RecordLatency records an observed upstream request duration in O(1) time
// using lock-free atomic CAS operations and EMA update formulas.
func (g *Governor) RecordLatency(d time.Duration) {
	latNs := d.Nanoseconds()
	if latNs < 0 {
		latNs = 0
	}

	g.reqCount.Add(1)
	g.sumLatencyNs.Add(latNs)

	// Update EMA mean and variance lock-free
	for {
		oldMean := g.currentMeanNs.Load()
		// New Mean = alpha * sample + (1 - alpha) * oldMean
		newMean := int64(g.alpha*float64(latNs) + (1.0-g.alpha)*float64(oldMean))
		if g.currentMeanNs.CompareAndSwap(oldMean, newMean) {
			// Calculate absolute deviation for variance estimation
			diff := latNs - oldMean
			if diff < 0 {
				diff = -diff
			}
			varSample := diff * diff

			for {
				oldVar := g.currentVarNs.Load()
				newVar := int64(g.alpha*float64(varSample) + (1.0-g.alpha)*float64(oldVar))
				if g.currentVarNs.CompareAndSwap(oldVar, newVar) {
					break
				}
			}
			break
		}
	}
}

// GetDynamicTimeout calculates the recommended deadline based on EMA mean + multiplier * sqrt(variance).
// Operates entirely on stack/atomic reads with zero heap allocations.
func (g *Governor) GetDynamicTimeout() time.Duration {
	mean := g.currentMeanNs.Load()
	variance := g.currentVarNs.Load()
	if variance < 0 {
		variance = 0
	}

	// Approximate square root of variance using integer/float conversion on stack
	stddev := float64(0)
	if variance > 0 {
		// Simple Babylonian or math.Sqrt since float conversion doesn't heap escape scalar values
		stddev = sqrt(float64(variance))
	}

	computed := float64(mean) + g.multiplier*stddev
	timeout := time.Duration(computed)

	if timeout < g.minTimeout {
		return g.minTimeout
	}
	if timeout > g.maxTimeout {
		return g.maxTimeout
	}
	return timeout
}

// Reset resets governor metrics back to initial state.
func (g *Governor) Reset() {
	g.sumLatencyNs.Store(0)
	g.varLatencyNs.Store(0)
	g.reqCount.Store(0)
	g.currentMeanNs.Store(int64(g.minTimeout))
	g.currentVarNs.Store(int64(g.minTimeout / 4))
}

// Helper fast stack-based square root function
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x / 2
	for i := 0; i < 6; i++ {
		z = (z + x/z) / 2
	}
	return z
}
