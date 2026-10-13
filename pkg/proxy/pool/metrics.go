package pool

import "sync/atomic"

// MetricsSnapshot represents a point-in-time view of the governor state.
type MetricsSnapshot struct {
	DialSuccess   uint64
	DialFailure   uint64
	ReuseCount    uint64
	SocketExhaust uint64
}

// GetSnapshot extracts metrics without heap allocations.
func (g *Governor) GetSnapshot() MetricsSnapshot {
	return MetricsSnapshot{
		DialSuccess:   atomic.LoadUint64(&g.DialSuccess[0]),
		DialFailure:   atomic.LoadUint64(&g.DialFailure[0]),
		ReuseCount:    atomic.LoadUint64(&g.ReuseCount[0]),
		SocketExhaust: atomic.LoadUint64(&g.SocketExhaust[0]),
	}
}
