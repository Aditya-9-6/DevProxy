package waf

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrRateLimitExceeded = errors.New("waf: rate limit exceeded")
	ErrContextCancelled  = errors.New("waf: context cancelled during rate limit check")
)

// CacheLinePad represents cache-line padding (64 bytes) to prevent false sharing
// across CPU cores in high-throughput lock-free operations.
type CacheLinePad struct {
	_ [64]byte
}

// BucketState holds the atomic token bucket state packed into a single word
// or tightly controlled atomic fields with cache padding.
type BucketState struct {
	// LastUpdateNano is the Unix nano timestamp of the last token replenishment.
	LastUpdateNano int64
	// Tokens is the current available token count represented as bits or scaled integers.
	Tokens int64
	_      CacheLinePad
}

// TokenBucket implements a lock-free cache-aligned token bucket rate limiter.
type TokenBucket struct {
	capacity   int64
	refillRate int64 // tokens per second
	state      BucketState
	_          CacheLinePad
}

var bucketPool = sync.Pool{
	New: func() any {
		return &TokenBucket{}
	},
}

// NewTokenBucket creates a new lock-free token bucket with specified capacity and refill rate.
func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
	b := bucketPool.Get().(*TokenBucket)
	b.capacity = capacity
	b.refillRate = refillRate
	b.state.Tokens = capacity
	b.state.LastUpdateNano = time.Now().UnixNano()
	return b
}

// Release returns the token bucket back to the sync pool.
func (b *TokenBucket) Release() {
	bucketPool.Put(b)
}

// Allow attempts to consume 'tokens' from the bucket using atomic CAS operations.
// It executes with zero heap allocations and sub-microsecond latency.
func (b *TokenBucket) Allow(ctx context.Context, requestedTokens int64) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, ErrContextCancelled
	}

	if requestedTokens <= 0 {
		return true, nil
	}

	if requestedTokens > b.capacity {
		return false, nil
	}

	for {
		select {
		case <-ctx.Done():
			return false, ErrContextCancelled
		default:
		}

		lastUpdate := atomic.LoadInt64(&b.state.LastUpdateNano)
		currentTokens := atomic.LoadInt64(&b.state.Tokens)

		now := time.Now().UnixNano()
		elapsedNano := now - lastUpdate

		var newTokens int64
		if elapsedNano > 0 {
			// Calculate replenished tokens without floating-point math for zero allocation & speed
			refilled := (elapsedNano * b.refillRate) / int64(time.Second)
			newTokens = currentTokens + refilled
			if newTokens > b.capacity {
				newTokens = b.capacity
			}
		} else {
			newTokens = currentTokens
		}

		if newTokens < requestedTokens {
			return false, nil
		}

		remainingTokens := newTokens - requestedTokens

		// Attempt atomic CAS update on token state and timestamp
		if atomic.CompareAndSwapInt64(&b.state.LastUpdateNano, lastUpdate, now) {
			if atomic.CompareAndSwapInt64(&b.state.Tokens, currentTokens, remainingTokens) {
				return true, nil
			}
			// Rollback or retry if token CAS failed due to concurrent modification
		}
	}
}
