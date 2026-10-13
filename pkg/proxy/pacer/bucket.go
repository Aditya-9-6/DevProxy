package pacer

import (
	"sync/atomic"
	"time"
)

// CacheLinePad prevents false sharing on CPU caches.
type CacheLinePad [8]uint64

// TokenBucket implements a lock-free packet pacer.
type TokenBucket struct {
	capacity   uint64
	refillRate uint64
	lastRefill atomic.Uint64
	tokens     atomic.Uint64
	_pad       CacheLinePad
}

func NewTokenBucket(rate, capacity uint64) *TokenBucket {
	b := &TokenBucket{
		capacity:   capacity,
		refillRate: rate,
	}
	b.tokens.Store(capacity)
	b.lastRefill.Store(uint64(time.Now().UnixNano()))
	return b
}

func (b *TokenBucket) Consume(n uint64) bool {
	for {
		b.refill()
		current := b.tokens.Load()
		if current < n {
			return false
		}
		if b.tokens.CompareAndSwap(current, current-n) {
			return true
		}
	}
}

func (b *TokenBucket) refill() {
	now := uint64(time.Now().UnixNano())
	last := b.lastRefill.Load()

	delta := now - last
	if delta < 1e6 { // Refill at most every 1ms
		return
	}

	if b.lastRefill.CompareAndSwap(last, now) {
		refillAmount := (delta * b.refillRate) / 1e9
		if refillAmount > 0 {
			for {
				old := b.tokens.Load()
				new := old + refillAmount
				if new > b.capacity {
					new = b.capacity
				}
				if b.tokens.CompareAndSwap(old, new) {
					break
				}
			}
		}
	}
}
