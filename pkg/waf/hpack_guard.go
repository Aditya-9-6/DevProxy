package waf

import (
	"sync/atomic"
)

// HPACKGuard enforces memory bounds on dynamic table state to prevent decompression bombs.
type HPACKGuard struct {
	maxTableSize uint64
	currentSize  atomic.Uint64
}

// NewHPACKGuard initializes a guard with a specific memory limit in bytes.
func NewHPACKGuard(maxBytes uint64) *HPACKGuard {
	return &HPACKGuard{maxTableSize: maxBytes}
}

// ValidateEntry checks if adding an entry of size 'n' exceeds the allowed limit.
func (g *HPACKGuard) ValidateEntry(n uint64) bool {
	for {
		old := g.currentSize.Load()
		if old+n > g.maxTableSize {
			return false
		}
		if g.currentSize.CompareAndSwap(old, old+n) {
			return true
		}
	}
}

// ReleaseEntry decrements the tracked size after an entry is evicted.
func (g *HPACKGuard) ReleaseEntry(n uint64) {
	for {
		old := g.currentSize.Load()
		if n > old {
			g.currentSize.Store(0)
			return
		}
		if g.currentSize.CompareAndSwap(old, old-n) {
			return
		}
	}
}

// GetCurrentSize returns the current atomic size of the dynamic table.
func (g *HPACKGuard) GetCurrentSize() uint64 {
	return g.currentSize.Load()
}
