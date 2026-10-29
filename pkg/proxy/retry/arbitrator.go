package retry

import (
	"sync/atomic"
)

type StreamState int32

const (
	StreamIdle StreamState = iota
	StreamHeadersVisible
	StreamBodyVisible
	StreamCommitted
)

// Arbitrator evaluates whether a request stream can be safely retried.
type Arbitrator struct {
	state int32 // StreamState
}

func NewArbitrator() *Arbitrator {
	return &Arbitrator{state: int32(StreamIdle)}
}

func (a *Arbitrator) State() StreamState {
	return StreamState(atomic.LoadInt32(&a.state))
}

func (a *Arbitrator) MarkHeadersVisible() {
	atomic.CompareAndSwapInt32(&a.state, int32(StreamIdle), int32(StreamHeadersVisible))
}

func (a *Arbitrator) MarkBodyVisible() {
	atomic.CompareAndSwapInt32(&a.state, int32(StreamHeadersVisible), int32(StreamBodyVisible))
	atomic.CompareAndSwapInt32(&a.state, int32(StreamIdle), int32(StreamBodyVisible))
}

func (a *Arbitrator) MarkCommitted() {
	atomic.StoreInt32(&a.state, int32(StreamCommitted))
}

func (a *Arbitrator) CanRetry(replaySafe bool) bool {
	if !replaySafe {
		return false
	}
	st := a.State()
	// Only retry if uncommitted and headers/body have not been exposed downstream
	if st == StreamCommitted || st == StreamBodyVisible {
		return false
	}
	return true
}
