package relay

import (
	"sync/atomic"
)

// DirectionState represents the state of one half of a duplex stream.
type DirectionState uint32

const (
	StateActive DirectionState = iota
	StateHalfClosed
	StateClosed
)

// DuplexState manages coordinated two-way shutdown and termination states.
type DuplexState struct {
	clientToServer uint32
	serverToClient uint32
	closed         uint32
}

// NewDuplexState creates an initialized active duplex state.
func NewDuplexState() *DuplexState {
	return &DuplexState{
		clientToServer: uint32(StateActive),
		serverToClient: uint32(StateActive),
		closed:         0,
	}
}

// SetHalfClosed marks one direction as half-closed on EOF.
func (s *DuplexState) SetHalfClosed(clientToServer bool) bool {
	if clientToServer {
		atomic.StoreUint32(&s.clientToServer, uint32(StateHalfClosed))
	} else {
		atomic.StoreUint32(&s.serverToClient, uint32(StateHalfClosed))
	}
	return s.IsTerminated()
}

// IsTerminated returns true when both directions are finished or session was closed.
func (s *DuplexState) IsTerminated() bool {
	if atomic.LoadUint32(&s.closed) == 1 {
		return true
	}
	c2s := atomic.LoadUint32(&s.clientToServer)
	s2c := atomic.LoadUint32(&s.serverToClient)
	return (c2s == uint32(StateHalfClosed) || c2s == uint32(StateClosed)) &&
		(s2c == uint32(StateHalfClosed) || s2c == uint32(StateClosed))
}

// MarkClosed atomically marks the entire session closed.
func (s *DuplexState) MarkClosed() bool {
	return atomic.CompareAndSwapUint32(&s.closed, 0, 1)
}
