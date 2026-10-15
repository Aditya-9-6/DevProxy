package quic

import (
	"sync/atomic"
)

// WindowManager manages flow-control credits for QUIC streams.
// It uses atomic operations to ensure lock-free updates to window sizes.
type WindowManager struct {
	// availableWindow tracks the current credit available for transmission.
	availableWindow CacheLinePad
	// maxWindow tracks the maximum allowed window size.
	maxWindow uint64
}

// NewWindowManager initializes a new flow control window manager.
func NewWindowManager(initialWindow uint64) *WindowManager {
	wm := &WindowManager{
		maxWindow: initialWindow,
	}
	atomic.StoreUint64(&wm.availableWindow[0], initialWindow)
	return wm
}

// Consume attempts to reserve 'n' bytes from the window.
// Returns true if successful, false otherwise.
func (wm *WindowManager) Consume(n uint64) bool {
	for {
		current := atomic.LoadUint64(&wm.availableWindow[0])
		if current < n {
			return false
		}
		if atomic.CompareAndSwapUint64(&wm.availableWindow[0], current, current-n) {
			return true
		}
	}
}

// Replenish adds 'n' bytes back to the available window.
func (wm *WindowManager) Replenish(n uint64) {
	for {
		current := atomic.LoadUint64(&wm.availableWindow[0])
		newVal := current + n
		if newVal > wm.maxWindow {
			newVal = wm.maxWindow
		}
		if atomic.CompareAndSwapUint64(&wm.availableWindow[0], current, newVal) {
			return
		}
	}
}

// Available returns the current window size.
func (wm *WindowManager) Available() uint64 {
	return atomic.LoadUint64(&wm.availableWindow[0])
}
