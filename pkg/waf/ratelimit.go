package waf

import (
	"sync"
	"time"
)

// TokenBucket implements a thread-safe rate limiter using a mutex.
// This ensures atomic updates to both token count and refill timestamp.
type TokenBucket struct {
	mu       sync.Mutex
	capacity uint64
	refillNs uint64
	tokens   uint64
	last     uint64
}

// NewTokenBucket creates a new rate limiter.
func NewTokenBucket(ratePerSec uint64, burst uint64) *TokenBucket {
	return &TokenBucket{
		capacity: burst,
		refillNs: uint64(time.Second.Nanoseconds()) / ratePerSec,
		tokens:   burst,
		last:     uint64(time.Now().UnixNano()),
	}
}

// Allow attempts to consume a token. Returns true if allowed.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := uint64(time.Now().UnixNano())
	if now < tb.last {
		now = tb.last // Handle clock drift
	}

	elapsed := now - tb.last
	refill := elapsed / tb.refillNs

	if refill > 0 {
		tb.tokens += refill
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.last = now - (elapsed % tb.refillNs)
	}

	if tb.tokens == 0 {
		return false
	}

	tb.tokens--
	return true
}
