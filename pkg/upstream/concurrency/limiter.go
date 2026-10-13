package concurrency

import (
	"sync/atomic"
	"time"
)

type Limiter struct {
	limit    atomic.Int64
	inflight atomic.Int64
	tracker  *LatencyTracker
	minLimit int64
	maxLimit int64
}

func NewLimiter(min, max int64) *Limiter {
	l := &Limiter{
		minLimit: min,
		maxLimit: max,
		tracker:  &LatencyTracker{},
	}
	l.limit.Store(min)
	return l
}

func (l *Limiter) Acquire() bool {
	for {
		curr := l.inflight.Load()
		if curr >= l.limit.Load() {
			return false
		}
		if l.inflight.CompareAndSwap(curr, curr+1) {
			return true
		}
	}
}

func (l *Limiter) Release(latency time.Duration) {
	l.inflight.Add(-1)
	l.tracker.Record(latency)
	// Simple gradient adjustment: if p95 > 500ms, decrease limit
	if l.tracker.GetPercentile(0.95) > 500*time.Millisecond {
		newLimit := l.limit.Load() - 1
		if newLimit < l.minLimit {
			newLimit = l.minLimit
		}
		l.limit.Store(newLimit)
	} else {
		newLimit := l.limit.Load() + 1
		if newLimit > l.maxLimit {
			newLimit = l.maxLimit
		}
		l.limit.Store(newLimit)
	}
}
