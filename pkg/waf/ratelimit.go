package waf

import (
	"sync/atomic"
	"time"
)

// CacheLinePad prevents false sharing by padding atomic counters to 64 bytes.
type CacheLinePad [8]uint64

// TokenBucket implements a lock-free, cache-aligned rate limiter.
// It uses atomic CAS to manage token state without mutex contention.
type TokenBucket struct {
	capacity uint64
	refillNs uint64
	state    CacheLinePad // [0]: tokens, [1]: lastRefillNs
}

// NewTokenBucket creates a new rate limiter.
func NewTokenBucket(ratePerSec uint64, burst uint64) *TokenBucket {
	return &TokenBucket{
		capacity: burst,
		refillNs: uint64(time.Second.Nanoseconds()) / ratePerSec,
	}
}

// Allow attempts to consume a token. Returns true if allowed.
func (tb *TokenBucket) Allow() bool {
	for {
		now := uint64(time.Now().UnixNano())
		tokens := atomic.LoadUint64(&tb.state[0])
		last := atomic.LoadUint64(&tb.state[1])

		// Calculate replenishment
		elapsed := now - last
		refill := elapsed / tb.refillNs

		newTokens := tokens
		if refill > 0 {
			newTokens += refill
			if newTokens > tb.capacity {
				newTokens = tb.capacity
			}
		}

		if newTokens == 0 {
			return false
		}

		// Attempt CAS update
		nextTokens := newTokens - 1
		nextLast := last
		if refill > 0 {
			nextLast = now - (elapsed % tb.refillNs)
		}

		if atomic.CompareAndSwapUint64(&tb.state[0], tokens, nextTokens) {
			atomic.StoreUint64(&tb.state[1], nextLast)
			return true
		}
	}
}
