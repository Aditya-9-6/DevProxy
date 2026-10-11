package h2

import (
	"sync/atomic"
)

// Window manages flow control credits using atomic operations for lock-free updates.
type Window struct {
	credits atomic.Int64
}

func NewWindow(initial int64) *Window {
	w := &Window{}
	w.credits.Store(initial)
	return w
}

// Consume attempts to deduct credits. Returns true if successful.
func (w *Window) Consume(amount int64) bool {
	for {
		current := w.credits.Load()
		if current < amount {
			return false
		}
		if w.credits.CompareAndSwap(current, current-amount) {
			return true
		}
	}
}

// Replenish adds credits to the window.
func (w *Window) Replenish(amount int64) {
	w.credits.Add(amount)
}

// Get returns current available credits.
func (w *Window) Get() int64 {
	return w.credits.Load()
}
