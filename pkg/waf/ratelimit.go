package waf

import (
	"context"
	"errors"
	"sync"
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

// BucketState holds the atomic token bucket state protected by a mutex
// or packed representation to guarantee atomic updates of LastUpdateNano and Tokens.
type BucketState struct {
	mu             sync.Mutex
	LastUpdateNano int64
	Tokens         int64
	_              CacheLinePad
}

// TokenBucket implements a cache-aligned token bucket rate limiter with atomic state protection.
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

// NewTokenBucket creates a new token bucket with specified capacity and refill rate.
func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
	b := bucketPool.Get().(*TokenBucket)
	b.capacity = capacity
	b.refillRate = refillRate
	b.state.mu.Lock()
	b.state.Tokens = capacity
	b.state.LastUpdateNano = time.Now().UnixNano()
	b.state.mu.Unlock()
	return b
}

// Release returns the token bucket back to the sync pool.
func (b *TokenBucket) Release() {
	bucketPool.Put(b)
}

// Allow attempts to consume 'tokens' from the bucket using synchronized critical section updates
// to prevent split-brain state issues between timestamp and token count.
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

		b.state.mu.Lock()
		now := time.Now().UnixNano()
		lastUpdate := b.state.LastUpdateNano
		currentTokens := b.state.Tokens

		granularityElapsed := now - lastUpdate
		var newTokens int64
		if granularityElapsed > 0 {
			refilled := (granularityElapsed * b.refillRate) / int64(time.Second)
			newTokens = currentTokens + refilled
			if newTokens > b.capacity {
				newTokens = b.capacity
			}
		} else {
			newTokens = currentTokens
		}

		dataOk := newTokens >= requestedTokens
		if !dataOk {
			b.state.mu.Unlock()
			return false, nil
		}

		remainingTokens := newTokens - requestedTokens
		b.state.LastUpdateNano = now
		b.state.Tokens = remainingTokens
		b.state.mu.Unlock()
		return true, nil
	}
}
