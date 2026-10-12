package proxy

import (
	"sync/atomic"
	"time"
)

const cacheLinePad = 64

type paddedCounter struct {
	count atomic.Uint64
	_     [cacheLinePad - 8]byte
}

type TokenBucket struct {
	capacity   uint64
	refillRate uint64
	lastRefill atomic.Int64
	tokens     paddedCounter
}

func NewTokenBucket(capacity, refillRate uint64) *TokenBucket {
	b := &TokenBucket{
		capacity:   capacity,
		refillRate: refillRate,
	}
	b.tokens.count.Store(capacity)
	b.lastRefill.Store(time.Now().UnixNano())
	return b
}

func (b *TokenBucket) Allow() bool {
	now := time.Now().UnixNano()
	last := b.lastRefill.Load()

	delta := uint64(now - last)
	if delta > 0 {
		refill := (delta * b.refillRate) / uint64(time.Second)
		if refill > 0 {
			if b.lastRefill.CompareAndSwap(last, now) {
				current := b.tokens.count.Load()
				for {
					newVal := current + refill
					if newVal > b.capacity {
						newVal = b.capacity
					}
					if b.tokens.count.CompareAndSwap(current, newVal) {
						break
					}
					current = b.tokens.count.Load()
				}
			}
		}
	}

	for {
		current := b.tokens.count.Load()
		if current == 0 {
			return false
		}
		if b.tokens.count.CompareAndSwap(current, current-1) {
			return true
		}
	}
}
