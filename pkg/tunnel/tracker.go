package tunnel

import "sync/atomic"

// ConnectionTracker provides lock-free tracking of active tunnel streams.
type ConnectionTracker struct {
	activeStreams atomic.Uint64
}

func (t *ConnectionTracker) Increment() {
	t.activeStreams.Add(1)
}

func (t *ConnectionTracker) Decrement() {
	t.activeStreams.Add(^uint64(0))
}

func (t *ConnectionTracker) ActiveCount() uint64 {
	return t.activeStreams.Load()
}
