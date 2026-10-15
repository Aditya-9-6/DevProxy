package splice

import "sync/atomic"

// PipePair represents a kernel pipe used for splice(2) operations.
type PipePair struct {
	Reader int
	Writer int
}

// Pool manages a set of pre-allocated pipes to avoid syscall overhead.
type Pool struct {
	pipes    []PipePair
	mask     atomic.Uint64
	capacity int
}
