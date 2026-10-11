package proxy

import (
	"testing"
	"time"
)

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	if !cb.Allow() {
		t.Error("Expected allowed in closed state")
	}

	cb.RecordFailure()
	cb.RecordFailure()

	if cb.Allow() {
		t.Error("Expected blocked in open state")
	}

	time.Sleep(150 * time.Millisecond)

	if !cb.Allow() {
		t.Error("Expected allowed in half-open state after timeout")
	}

	cb.RecordSuccess()
	if cb.state.Load() != int32(StateClosed) {
		t.Error("Expected closed after success in half-open")
	}
}
