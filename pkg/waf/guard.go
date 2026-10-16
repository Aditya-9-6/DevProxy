package waf

import (
	"sync/atomic"
)

// ReDoSGuard tracks regex execution steps to prevent catastrophic backtracking.
type ReDoSGuard struct {
	MaxSteps uint64
	Steps    CacheLinePad
}

func NewReDoSGuard(maxSteps uint64) *ReDoSGuard {
	return &ReDoSGuard{MaxSteps: maxSteps}
}

func (g *ReDoSGuard) Increment() bool {
	val := atomic.AddUint64(&g.Steps[0], 1)
	return val <= g.MaxSteps
}

func (g *ReDoSGuard) Reset() {
	atomic.StoreUint64(&g.Steps[0], 0)
}
