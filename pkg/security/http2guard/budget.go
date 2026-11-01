package http2guard

import (
	"sync/atomic"
	"time"
)

// Budget tracks atomics-based limits for rapid resets and SETTINGS churn.
type Budget struct {
	maxRapidResets int64
	maxSettings    int64
	resetTokens    atomic.Int64
	settingsTokens atomic.Int64
	lastRefill     atomic.Int64 // Unix nano timestamp
	refillInterval int64        // Nano interval for token regeneration
}

func NewBudget(maxRapidResets, maxSettings int64, refillWindow time.Duration) *Budget{
	b := &Budget{
		maxRapidResets: maxRapidResets,
		maxSettings:    maxSettings,
		refillInterval: refillWindow.Nanoseconds(),
	}
	b.resetTokens.Store(maxRapidResets)
	b.settingsTokens.Store(maxSettings)
	b.lastRefill.Store(time.Now().UnixNano())
	return b
}

func (b *Budget) refill(nowNano int64) {
	last := b.lastRefill.Load()
	interval := b.refillInterval
	if interval <= 0 {
		return
	}
	if nowNano-last < interval {
		return
	}
	if b.lastRefill.CompareAndSwap(last, nowNano) {
		b.resetTokens.Store(b.maxRapidResets)
		b.settingsTokens.Store(b.maxSettings)
	}
}

func (b *Budget) ConsumeReset(nowNano int64) bool {
	b.refill(nowNano)
	for {
		current := b.resetTokens.Load()
		if current <= 0 {
			return false
		}
		if b.resetTokens.CompareAndSwap(current, current-1) {
			return true
		}
	}
}

func (b *Budget) ConsumeSettings(nowNano int64) bool {
	b.refill(nowNano)
	for {
		current := b.settingsTokens.Load()
		if current <= 0 {
			return false
		}
		if b.settingsTokens.CompareAndSwap(current, current-1) {
			return true
		}
	}
}
