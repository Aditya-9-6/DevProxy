package proxy

import (
	"sync/atomic"
	"time"
)

// TokenBucket implements a lock-free rate limiter.
type TokenBucket struct {
	capacity   uint64
	refillRate uint64
	lastRefill atomic.Int64
	tokens     atomic.Uint64
}

// NewTokenBucket creates a new bucket with capacity and tokens per second.
func NewTokenBucket(capacity, rate uint64) *TokenBucket {
	b := &TokenBucket{
		capacity:   capacity,
		refillRate: rate,
	}
	b.tokens.Store(capacity)
	b.lastRefill.Store(time.Now().UnixNano())
	return b
}

// Allow checks if a token can be consumed.
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
			b.lastRefill.Store(now)
			return true
		}
	}
}
