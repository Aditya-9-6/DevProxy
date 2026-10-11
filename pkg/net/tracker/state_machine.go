package tracker

import (
	"sync/atomic"
)

// ConnectionState represents the TCP half-open state.
type ConnectionState uint32

const (
	StateIdle ConnectionState = iota
	StateSynSent
	StateSynReceived
	StateEstablished
)

// CacheLinePad prevents false sharing by padding the struct to 64 bytes.
type CacheLinePad struct {
	_ [56]byte
}

// TrackedState wraps an atomic state with padding.
type TrackedState struct {
	state atomic.Uint32
	_     CacheLinePad
}

// StateMachine manages a fixed-size pool of connection states.
type StateMachine struct {
	states []TrackedState
	size   uint32
}

// NewStateMachine initializes a tracker with a fixed number of slots.
func NewStateMachine(size uint32) *StateMachine {
	return &StateMachine{
		states: make([]TrackedState, size),
		size:   size,
	}
}

// Update attempts to transition the state of a connection slot using CAS.
func (sm *StateMachine) Update(index uint32, old, new ConnectionState) bool {
	if index >= sm.size {
		return false
	}
	return sm.states[index].state.CompareAndSwap(uint32(old), uint32(new))
}

// Get returns the current state of a connection slot.
func (sm *StateMachine) Get(index uint32) ConnectionState {
	if index >= sm.size {
		return StateIdle
	}
	return ConnectionState(sm.states[index].state.Load())
}

// Reset clears a slot back to idle.
func (sm *StateMachine) Reset(index uint32) {
	if index < sm.size {
		sm.states[index].state.Store(uint32(StateIdle))
	}
}
