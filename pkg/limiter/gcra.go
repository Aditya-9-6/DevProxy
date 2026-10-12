package limiter

import (
	"sync"
	"sync/atomic"
	"time"
)

// GCRALimiter implements the Generic Cell Rate Algorithm for L7 rate limiting
// using a lock-free atomic Theoretical Arrival Time (TAT) tracker per key.
// It includes cache-line padding (64 bytes) to prevent false sharing across CPU cores.
type GCRALimiter struct {
	// Padding to avoid false sharing on cache lines
	_ [8]uint64

	// emissionInterval is the interval between valid cell arrivals (nanoseconds)
	emissionInterval int64
	// burstLimit is the maximum tolerance for burstiness (nanoseconds)
	burstLimit int64

	mu   sync.RWMutex
	tats map[string]*paddedTAT
}

// paddedTAT wraps an atomic.Int64 with 64-byte padding to prevent false sharing.
type paddedTAT struct {
	_   [8]uint64
	tat atomic.Int64
	_   [8]uint64
}

// NewGCRALimiter creates a GCRALimiter with a given rate (requests per second) and burst size.
func NewGCRALimiter(rate float64, burst int) *GCRALimiter {
	if rate <= 0 {
		rate = 1.0
	}
	if burst <= 0 {
		oburst := int(rate)
		if oburst < 1 {
			oburst = 1
		}
		burst = oburst
	}

	// emission interval = 1 second / rate (in nanoseconds)
	emissionInterval := int64(float64(time.Second) / rate)
	// burst limit = emission interval * burst size
	burstLimit := emissionInterval * int64(burst)

	return &GCRALimiter{
		emissionInterval: emissionInterval,
		burstLimit:       burstLimit,
		tats:             make(map[string]*paddedTAT),
	}
}

// Allow checks whether a request associated with key can proceed under the GCRA parameters.
// Returns true if allowed, false if rate limited.
func (l *GCRALimiter) Allow(key string) bool {
	now := time.Now().UnixNano()

	l.mu.RLock()
	p, exists := l.tats[key]
	l.mu.RUnlock()

	if !exists {
		l.mu.Lock()
		p, exists = l.tats[key]
		if !exists {
			p = &paddedTAT{}
			p.tat.Store(now)
			l.tats[key] = p
		}
		l.mu.Unlock()
	}

	// GCRA algorithm implementation with atomic CAS loop on TAT
	for {
		tat := p.tat.Load()
		valTime := tat
		if valTime < now {
			valTime = now
		}

		// Calculate theoretical arrival time for the next cell
		newTAT := valTime + l.emissionInterval
		// Check if the arrival is too far in the future (exceeding burst limit)
		if newTAT-now > l.burstLimit {
			return false
		}

		// Try to atomically update the TAT
		if p.tat.CompareAndSwap(tat, newTAT) {
			return true
		}
	}
}

// Reset clears all tracked limits.
func (l *GCRALimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tats = make(map[string]*paddedTAT)
}
