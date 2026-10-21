package waf

import (
	"sync"
	"time"
)

// RateLimiter is a token bucket rate limiter that can be used to throttle
// requests per identifier. It is safe for concurrent use.
type RateLimiter struct {
	mu         sync.Mutex
	capacity   uint32
	refillRate uint32
	refillNs   int64
	tokens     float64
	lastRefill time.Time
}

// NewRateLimiter creates a new RateLimiter with the given capacity and refill
// rate. refillNs is the interval in nanoseconds at which tokens are refilled.
func NewRateLimiter(capacity, refillRate uint32, refillNs int64) *RateLimiter {
	return &RateLimiter{
		capacity:   capacity,
		refillRate: refillRate,
		refillNs:   refillNs,
		tokens:     float64(capacity),
		lastRefill: time.Now(),
	}
}

// Allow reports whether a request with the given identifier is allowed.
// The identifier is currently ignored; the rate limiter is global.
func (r *RateLimiter) Allow(id int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Nanoseconds()
	if r.refillRate > 0 && elapsed > r.refillNs {
		intervals := float64(elapsed) / float64(r.refillNs)
		addedTokens := intervals * float64(r.refillRate)
		r.tokens += addedTokens
		if r.tokens > float64(r.capacity) {
			r.tokens = float64(r.capacity)
		}
		r.lastRefill = now
	}

	if r.tokens >= 1.0 {
		r.tokens -= 1.0
		return true
	}
	return false
}
