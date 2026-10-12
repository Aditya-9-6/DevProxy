package proxy

import (
	"sync/atomic"
	"time"
)

// TokenBucket implements a thread-safe, lock-free token bucket.
type TokenBucket struct {
	capacity   uint64
	refillRate uint64
	state      atomic.Pointer[bucketState]
}

type bucketState struct {
	tokens     uint64
	lastRefill int64
}

func NewTokenBucket(capacity, refillRate uint64) *TokenBucket {
	b := &TokenBucket{
		capacity:   capacity,
		refillRate: refillRate,
	}
	b.state.Store(&bucketState{tokens: capacity, lastRefill: time.Now().UnixNano()})
	return b
}

func (b *TokenBucket) Allow() bool {
	for {
		old := b.state.Load()
		now := time.Now().UnixNano()

		delta := now - old.lastRefill
		refill := uint64(0)
		if delta > 0 {
			refill = (uint64(delta) * b.refillRate) / uint64(time.Second)
		}

		newTokens := old.tokens + refill
		if newTokens > b.capacity {
			newTokens = b.capacity
		}

		if newTokens == 0 {
			return false
		}

		next := &bucketState{
			tokens:     newTokens - 1,
			lastRefill: now,
		}

		if b.state.CompareAndSwap(old, next) {
			return true
		}
	}
}
