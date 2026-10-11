package proxy

import (
	"sync"
	"time"
)

// rateWindow tracks timestamps for the sliding window rate limiter.
type rateWindow struct {
	timestamps []int64
	idx        int
}

// RateLimiter implements a sliding-window log rate limiter.
type RateLimiter struct {
	limit    uint64
	windowNs int64
	mu       sync.RWMutex
	buckets  map[string]*rateWindow
}

// NewRateLimiter creates a new sliding window rate limiter.
func NewRateLimiter(limit uint64, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:    limit,
		windowNs: window.Nanoseconds(),
		buckets:  make(map[string]*rateWindow),
	}
}

// Allow checks if the request is allowed under the rate limit.
func (rl *RateLimiter) Allow(key string) bool {
	now := time.Now().UnixNano()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	w, ok := rl.buckets[key]
	if !ok {
		w = &rateWindow{timestamps: make([]int64, rl.limit)}
		rl.buckets[key] = w
	}

	// Check if oldest entry is outside window
	count := 0
	for i := 0; i < int(rl.limit); i++ {
		if w.timestamps[i] > 0 && (now-w.timestamps[i]) < rl.windowNs {
			count++
		}
	}

	if uint64(count) >= rl.limit {
		return false
	}

	w.timestamps[w.idx] = now
	w.idx = (w.idx + 1) % int(rl.limit)
	return true
}
