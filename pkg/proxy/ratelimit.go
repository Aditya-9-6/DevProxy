package proxy

import (
	"sync/atomic"
	"time"
	// CacheLinePad is defined in canonical.go to avoid redeclaration conflicts.
)

// TokenBucket implements a thread-safe, lock-free token bucket using atomic fields.
// Padding is used to prevent false sharing on cache lines.
type TokenBucket struct {
	_          CacheLinePad
	capacity   uint64
	refillRate uint64
	_          CacheLinePad
	tokens     atomic.Uint64
	lastRefill atomic.Int64
	_          CacheLinePad
}

// NewTokenBucket creates a new token bucket with specified capacity and refill rate per second.
func NewTokenBucket(capacity, refillRate uint64) *TokenBucket {
	bp := &TokenBucket{
		capacity:   capacity,
		refillRate: refillRate,
	}
	bp.tokens.Store(capacity)
	bp.lastRefill.Store(time.Now().UnixNano())
	return bp
}

// Allow checks if a token can be consumed concurrently in a lock-free manner.
func (b *TokenBucket) Allow() bool {
	for {
		now := time.Now().UnixNano()
		last := b.lastRefill.Load()
		tok := b.tokens.Load()

		delta := now - last
		refill := uint64(0)
		if delta > 0 {
			refill = (uint64(delta) * b.refillRate) / uint64(time.Second)
		}

		newTokens := tok + refill
		if newTokens > b.capacity {
			newTokens = b.capacity
		}

		if newTokens == 0 {
			return false
		}

		if b.tokens.CompareAndSwap(tok, newTokens-1) {
			// Safely update lastRefill matching the tokens state transition
			b.lastRefill.Store(now)
			return true
		}
	}
}
