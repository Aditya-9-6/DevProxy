package pool

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to 64 bytes.
type CacheLinePad [8]uint64

// Governor tracks connection pool telemetry using lock-free atomics.
type Governor struct {
	DialSuccess   CacheLinePad
	DialFailure   CacheLinePad
	ReuseCount    CacheLinePad
	SocketExhaust CacheLinePad
}

// NewGovernor initializes a new telemetry governor.
func NewGovernor() *Governor {
	return &Governor{}
}

// RecordDialSuccess increments the success counter.
func (g *Governor) RecordDialSuccess() {
	atomic.AddUint64(&g.DialSuccess[0], 1)
}

// RecordDialFailure increments the failure counter.
func (g *Governor) RecordDialFailure() {
	atomic.AddUint64(&g.DialFailure[0], 1)
}

// RecordReuse increments the reuse counter.
func (g *Governor) RecordReuse() {
	atomic.AddUint64(&g.ReuseCount[0], 1)
}

// RecordSocketExhaust increments the exhaustion counter.
func (g *Governor) RecordSocketExhaust() {
	atomic.AddUint64(&g.SocketExhaust[0], 1)
}
