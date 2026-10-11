package proxy

import (
	"sync/atomic"
	"time"
)

type CircuitState int32

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	state           atomic.Int32
	failureCount    atomic.Uint64
	lastFailureTime atomic.Int64
	threshold       uint64
	resetTimeout    time.Duration
}

func NewCircuitBreaker(threshold uint64, resetTimeout time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		threshold:    threshold,
		resetTimeout: resetTimeout,
	}
	cb.state.Store(int32(StateClosed))
	return cb
}

func (cb *CircuitBreaker) Allow() bool {
	state := CircuitState(cb.state.Load())
	if state == StateClosed {
		return true
	}
	if state == StateOpen {
		if time.Now().UnixNano()-cb.lastFailureTime.Load() > cb.resetTimeout.Nanoseconds() {
			cb.state.CompareAndSwap(int32(StateOpen), int32(StateHalfOpen))
			return true
		}
		return false
	}
	return true
}

func (cb *CircuitBreaker) RecordSuccess() {
	if cb.state.Load() == int32(StateHalfOpen) {
		cb.failureCount.Store(0)
		cb.state.Store(int32(StateClosed))
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.lastFailureTime.Store(time.Now().UnixNano())
	count := cb.failureCount.Add(1)
	if count >= cb.threshold {
		cb.state.Store(int32(StateOpen))
	}
}
