package proxy

import (
	"sync"
	"testing"
	"time"
)

func TestCircuitBreakerInitialState(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(cfg)

	if cb.State() != StateClosed {
		t.Fatalf("expected initial state CLOSED, got %v", cb.State())
	}

	if err := cb.Allow(); err != nil {
		t.Fatalf("expected Allow() to succeed in CLOSED state, got %v", err)
	}
}

func TestCircuitBreakerTripAndHalfOpen(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold:  2,
		SuccessThreshold:  2,
		Timeout:           10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffFactor:     2.0,
		MaxProbes:         2,
	}
	cb := NewCircuitBreaker(cfg)

	// Record failures to trip circuit
	cb.Failure()
	if cb.State() != StateClosed {
		t.Fatalf("expected state CLOSED after 1 failure, got %v", cb.State())}

	cb.Failure()
	if cb.State() != StateOpen {
		t.Fatalf("expected state OPEN after 2 failures, got %v", cb.State())}

	// Attempt while open should fail
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}

	// Wait for timeout to elapse
	time.Sleep(15 * time.Millisecond)

	// Next Allow should transition to HALF_OPEN
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected Allow() to succeed in HALF_OPEN probe, got %v", err)
	}

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected state HALF_OPEN, got %v", cb.State())}

	// Record successes in HALF_OPEN to close circuit
	cb.Success()
	cb.Success()

	if cb.State() != StateClosed {
		t.Fatalf("expected state CLOSED after successful probes, got %v", cb.State())}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold:  50,
		SuccessThreshold:  10,
		Timeout:           5 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffFactor:     2.0,
		MaxProbes:         10,
	}
	cb := NewCircuitBreaker(cfg)

	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if err := cb.Allow(); err == nil {
					if (id+j)%2 == 0 {
						cb.Success()
					} else {
						cb.Failure()
					}
				}
			}
		}(i)
	}

	wg.Wait()
}
