package http2flow

import (
	"sync/atomic"
)

// Ledger tracks HTTP/2 flow control windows for a connection or stream using atomic primitives.
type Ledger struct {
	windowSize int64
	maxWindow  int64
}

// NewLedger initializes a flow ledger with an initial window size.
func NewLedger(initialWindow int64) *Ledger {
	return &Ledger{
		windowSize: initialWindow,
		maxWindow:  initialWindow,
	}
}

// Available returns the current available window size.
func (l *Ledger) Available() int64 {
	return atomic.LoadInt64(&l.windowSize)
}

// Consume attempts to deduct n bytes from the window. Returns false if window is exhausted.
func (l *Ledger) Consume(n int64) bool {
	for {
		current := atomic.LoadInt64(&l.windowSize)
		if current < n {
			return false
		}
		if atomic.CompareAndSwapInt64(&l.windowSize, current, current-n) {
			return true
		}
	}
}

// Replenish increases the window by n bytes, capping at maxWindow.
func (l *Ledger) Replenish(n int64) int64 {
	for {
		current := atomic.LoadInt64(&l.windowSize)
		newSize := current + n
		if newSize > l.maxWindow {
			newSize = l.maxWindow
		}
		if atomic.CompareAndSwapInt64(&l.windowSize, current, newSize) {
			return newSize
		}
	}
}

// ExpandMax increases the maximum capacity of the window.
func (l *Ledger) ExpandMax(delta int64) {
	atomic.AddInt64(&l.maxWindow, delta)
	l.Replenish(delta)
}
