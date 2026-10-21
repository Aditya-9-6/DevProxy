package http2guard

import (
	"fmt"
	"sync/atomic"
	"time"
)

const (
	cacheLineSize     = 64
	budgetCountBits   = 24
	budgetEpochBits   = 40
	budgetCounterMask = uint64(1<<budgetCountBits) - 1
	budgetEpochMask   = uint64(1<<budgetEpochBits) - 1
)

const (
	// MaxBurst is the largest number of events representable by a Budget.
	MaxBurst uint32 = (1 << budgetCountBits) - 1
	// MinimumWindow prevents generation counters from wrapping too quickly.
	MinimumWindow = time.Millisecond
)

// BudgetConfig defines a fixed-window burst budget.
// A zero Burst disables the budget.
type BudgetConfig struct {
	Burst  uint32
	Window time.Duration
}

func (c BudgetConfig) validate() error {
	if c.Burst > MaxBurst {
		return fmt.Errorf("burst limit %d exceeds maximum %d", c.Burst, MaxBurst)
	}
	if c.Window < 0 {
		return fmt.Errorf("budget window must not be negative: %s", c.Window)
	}
	if c.Burst > 0 && c.Window < MinimumWindow {
		return fmt.Errorf("budget window %s is below minimum %s", c.Window, MinimumWindow)
	}
	return nil
}

// paddedAtomicUint64 isolates each frequently-written counter from adjacent state.
type paddedAtomicUint64 struct {
	value uint64
	_     [cacheLineSize - 8]byte
}

// Budget is a lock-free fixed-window event budget.
// A Budget must not be copied after its first use.
type Budget struct {
	limit  uint32
	_      uint32
	window time.Duration
	state  paddedAtomicUint64
	_      [48]byte
}

func newBudget(config BudgetConfig) (Budget, error) {
	if err := config.validate(); err != nil {
		return Budget{}, err
	}
	return Budget{limit: config.Burst, window: config.Window}, nil
}

// NewBudget creates a lock-free fixed-window event budget.
func NewBudget(config BudgetConfig) (*Budget, error) {
	budget, err := newBudget(config)
	if err != nil {
		return nil, err
	}
	return &budget, nil
}

// Consume atomically charges one event and reports whether it fit the budget.
func (b *Budget) Consume(now time.Time) bool {
	if b == nil || b.limit == 0 {
		return true
	}

	epoch := budgetEpoch(now, b.window)
	for {
		current := atomic.LoadUint64(&b.state.value)
		currentEpoch := current >> budgetCountBits
		count := current & budgetCounterMask

		if currentEpoch != epoch {
			if atomic.CompareAndSwapUint64(&b.state.value, current, packBudgetState(epoch, 1)) {
				return true
			}
			continue
		}

		if count >= uint64(b.limit) {
			return false
		}
		if atomic.CompareAndSwapUint64(&b.state.value, current, packBudgetState(epoch, count+1)) {
			return true
		}
	}
}

// Count returns the number of charged events in the window containing now.
func (b *Budget) Count(now time.Time) uint32 {
	if b == nil || b.limit == 0 {
		return 0
	}

	state := atomic.LoadUint64(&b.state.value)
	if state>>budgetCountBits != budgetEpoch(now, b.window) {
		return 0
	}
	return uint32(state & budgetCounterMask)
}

// Reset clears the current counter. Normal window rollover also clears it.
func (b *Budget) Reset() {
	if b == nil {
		return
	}
	atomic.StoreUint64(&b.state.value, 0)
}

// Limit returns the configured maximum burst size.
func (b *Budget) Limit() uint32 {
	if b == nil {
		return 0
	}
	return b.limit
}

// Window returns the configured fixed-window duration.
func (b *Budget) Window() time.Duration {
	if b == nil {
		return 0
	}
	return b.window
}

func budgetEpoch(now time.Time, window time.Duration) uint64 {
	nanos := now.UnixNano()
	if nanos < 0 {
		nanos = 0
	}
	return (uint64(nanos) / uint64(window)) & budgetEpochMask
}

func packBudgetState(epoch uint64, count uint64) uint64 {
	return ((epoch & budgetEpochMask) << budgetCountBits) | (count & budgetCounterMask)
}
