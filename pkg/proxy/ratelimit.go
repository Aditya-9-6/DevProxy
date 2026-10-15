package proxy

import (
	"sync/atomic"
	"time"
)

// TokenBucketV2 implements a thread-safe, lock-free token bucket using atomic fields.
// Padding is used to prevent false sharing on cache lines.
type TokenBucketV2 struct {
	_          CacheLinePad
	capacity   uint64
	refillRate uint64
	_          CacheLinePad
	tokens     atomic.Uint64
	lastRefill atomic.Int64
	_          CacheLinePad
}

func NewTokenBucketV2(capacity, refillRate uint64) *TokenBucketV2 {
	b := &TokenBucketV2{
		capacity:   capacity,
		refillRate: refillRate,
	}
	b.tokens.Store(capacity)
	b.lastRefill.Store(time.Now().UnixNano())
	return b
}

func (b *TokenBucketV2) Allow() bool {
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
