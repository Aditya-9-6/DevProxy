package telemetry

import (
	"sync/atomic"
)

const (
	MaxBaggageItems = 1024
	CacheLineSize   = 64
)

// BaggageItem represents a single tracing key-value pair.
type BaggageItem struct {
	Key   string
	Value string
}

// BaggageBuffer implements a lock-free, cache-aligned ring buffer for tracing baggage.
type BaggageBuffer struct {
	_     [CacheLineSize]byte
	items [MaxBaggageItems]atomic.Pointer[BaggageItem]
	head  atomic.Uint64
	_     [CacheLineSize]byte
	tail  atomic.Uint64
	_     [CacheLineSize]byte
}

// NewBaggageBuffer initializes a new lock-free baggage ring buffer.
func NewBaggageBuffer() *BaggageBuffer {
	return &BaggageBuffer{}
}

// Push adds a new baggage item to the buffer, overwriting if full.
func (b *BaggageBuffer) Push(key, value string) {
	idx := b.tail.Add(1) % MaxBaggageItems
	item := &BaggageItem{Key: key, Value: value}
	b.items[idx].Store(item)
}

// Get retrieves all current items in the buffer.
func (b *BaggageBuffer) Get() []BaggageItem {
	var result []BaggageItem
	for i := 0; i < MaxBaggageItems; i++ {
		item := b.items[i].Load()
		if item != nil {
			result = append(result, *item)
		}
	}
	return result
}
