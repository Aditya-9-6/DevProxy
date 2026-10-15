package proxy

import (
	"errors"
	"sync/atomic"
)

// HeaderValidator provides lock-free, zero-allocation header dependency validation.
type HeaderValidator struct {
	state atomic.Uint64
}

// NewHeaderValidator creates a new validator instance.
func NewHeaderValidator() *HeaderValidator {
	return &HeaderValidator{}
}

// ObserveHeader updates the internal state machine with a header mask.
func (v *HeaderValidator) ObserveHeader(mask uint64) error {
	for {
		old := v.state.Load()
		new := old | mask

		// Check for dependency violations
		for _, rule := range DependencyTable {
			if (new & rule.Mask) == rule.Mask {
				return errors.New(rule.Err)
			}
		}

		if v.state.CompareAndSwap(old, new) {
			return nil
		}
	}
}

// Reset clears the validator state for reuse.
func (v *HeaderValidator) Reset() {
	v.state.Store(0)
}
