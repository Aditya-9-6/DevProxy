package proxy

import (
	"sync/atomic"
)

// SpliceMetrics implements lock-free cache-aligned atomic metrics for splice operations.
// Uses padding to prevent false sharing on hot paths.
type SpliceMetrics struct {
	_     [8]uint64
	count atomic.Uint64
	_     [8]uint64
	bytes atomic.Uint64
	_     [8]uint64
}

func NewSpliceMetrics() *SpliceMetrics {
	return &SpliceMetrics{}
}

func (t *SpliceMetrics) Record(bytes uint64) {
	t.count.Add(1)
	t.bytes.Add(bytes)
}

func (t *SpliceMetrics) Stats() (uint64, uint64) {
	return t.count.Load(), t.bytes.Load()
}
