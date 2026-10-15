package waf

import (
	"sync/atomic"
	"time"
)

// TokenBucket implements a lock-free, cache-aligned rate limiter using atomic CAS operations.
// It provides a high-performance token bucket algorithm for traffic shaping.
type TokenBucket struct {
	_        [56]byte // Padding for cache-line alignment
	capacity uint64
	refillNs uint64
	tokens   uint64
	last     uint64
	_        [56]byte // Padding for cache-line alignment
}

// NewTokenBucket creates a new rate limiter. If ratePerSec is 0, it defaults to 1.
func NewTokenBucket(ratePerSec uint64, burst uint64) *TokenBucket {
	if ratePerSec == 0 {
		ratePerSec = 1
	}
	return &TokenBucket{
		capacity: burst,
		refillNs: uint64(time.Second.Nanoseconds()) / ratePerSec,
		tokens:   burst,
		last:     uint64(time.Now().UnixNano()),
	}
}

// Allow attempts to consume a token using atomic CAS. Returns true if allowed.
func (tb *TokenBucket) Allow() bool {
	for {
		now := uint64(time.Now().UnixNano())
		last := atomic.LoadUint64(&tb.last)
		tokens := atomic.LoadUint64(&tb.tokens)

		if now < last {
			now = last
		}

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

		newTokens--
		newLast := now - (elapsed % tb.refillNs)

		// Attempt atomic update
		if atomic.CompareAndSwapUint64(&tb.tokens, tokens, newTokens) {
			atomic.StoreUint64(&tb.last, newLast)
			return true
		}
	}
}
