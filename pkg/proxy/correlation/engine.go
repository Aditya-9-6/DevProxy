package correlation

import (
	"sync/atomic"
	"unsafe"
)

// CacheLinePad prevents false sharing.
type CacheLinePad [8]uint64

// CorrelationEntry represents a single request-response mapping.
type CorrelationEntry struct {
	ID        uint64
	RequestID string
	Active    atomic.Bool
	Data      unsafe.Pointer
}

// Engine manages lock-free correlation of asynchronous streams.
type Engine struct {
	sequence atomic.Uint64
	pool     *Pool
}

func NewEngine() *Engine {
	return &Engine{
		pool: NewPool(),
	}
}

func (e *Engine) NextID() uint64 {
	return e.sequence.Add(1)
}

func (e *Engine) Acquire() *CorrelationEntry {
	return e.pool.Get()
}

func (e *Engine) Release(entry *CorrelationEntry) {
	entry.Active.Store(false)
	e.pool.Put(entry)
}
