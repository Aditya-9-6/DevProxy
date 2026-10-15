package quic

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to cache line boundaries.
type CacheLinePad [8]uint64

// CreditManager manages HTTP/3 datagram flow-control credits in a lock-free, cache-aligned manner.
type CreditManager struct {
	credits   CacheLinePad
	maxWindow CacheLinePad
	dropped   CacheLinePad
}

// NewCreditManager initializes a CreditManager with the given initial window and max capacity.
func NewCreditManager(initialWindow, maxWindow uint64) *CreditManager {
	m := &CreditManager{}
	atomic.StoreUint64(&m.credits[0], initialWindow)
	atomic.StoreUint64(&m.maxWindow[0], maxWindow)
	return m
}

// Acquire attempts to consume n credits. Returns true if successful, false if insufficient credit.
func (m *CreditManager) Acquire(n uint64) bool {
	for {
		current := atomic.LoadUint64(&m.credits[0])
		if current < n {
			atomic.AddUint64(&m.dropped[0], 1)
			return false
		}
		if atomic.CompareAndSwapUint64(&m.credits[0], current, current-n) {
			return true
		}
	}
}

// Release returns n credits back to the manager, capped at maxWindow.
func (m *CreditManager) Release(n uint64) {
	max := atomic.LoadUint64(&m.maxWindow[0])
	for {
		current := atomic.LoadUint64(&m.credits[0])
		next := current + n
		if next > max {
			next = max
		}
		if atomic.CompareAndSwapUint64(&m.credits[0], current, next) {
			break
		}
	}
}

// GetCredits returns the current available credit balance.
func (m *CreditManager) GetCredits() uint64 {
	return atomic.LoadUint64(&m.credits[0])
}

// GetDroppedCount returns the total number of blocked/dropped credit acquisition attempts.
func (m *CreditManager) GetDroppedCount() uint64 {
	return atomic.LoadUint64(&m.dropped[0])
}
