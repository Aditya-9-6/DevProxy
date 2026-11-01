package proxy

import (
	"errors"
	"sync/atomic"
	"time"
)

var (
	ErrCircuitOpen     = errors.New("proxy: circuit breaker is open")
	ErrMaxProbesExited = errors.New("proxy: maximum half-open probes exceeded")
)

// State represents the current operational mode of the circuit breaker.
type State uint32

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateHalfOpen:
		return "HALF_OPEN"
	case StateOpen:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig defines configuration parameters for the hysteresis circuit breaker.
type CircuitBreakerConfig struct {
	FailureThreshold    uint64        // Number of consecutive failures to transition from CLOSED to OPEN
	SuccessThreshold    uint64        // Number of consecutive successes in HALF_OPEN to transition to CLOSED
	Timeout             time.Duration // Base timeout duration before attempting a HALF_OPEN probe
	MaxBackoff          time.Duration // Maximum cap for exponential backoff duration
	BackoffFactor       float64       // Multiplier for exponential backoff (e.g., 2.0)
	MaxProbes           uint64        // Maximum concurrent or consecutive probes allowed in HALF_OPEN state
}

// DefaultCircuitBreakerConfig returns robust default settings for high-throughput proxies.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:  5,
		SuccessThreshold:  3,
		Timeout:           5 * time.Second,
		MaxBackoff:        60 * time.Second,
		BackoffFactor:     2.0,
		MaxProbes:         2,
	}
}

// CircuitBreaker implements a thread-safe, lock-free distributed hysteresis circuit breaker
// utilizing atomic primitives and exponential backoff probing.
type CircuitBreaker struct {
	state            uint32 // State
	failures         uint64 // Consecutive failure count
	successes        uint64 // Consecutive success count in half-open
	probeCount       uint64 // Active or completed probes in half-open
	nextAttemptNano  int64  // Unix nano timestamp when next probe is permitted
	currentBackoffNs int64  // Current backoff duration in nanoseconds
	config           CircuitBreakerConfig
}

// NewCircuitBreaker creates a new CircuitBreaker with the given configuration.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold == 0 {
		cfg.SuccessThreshold = 3
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.BackoffFactor <= 1.0 {
		cfg.BackoffFactor = 2.0
	}
	if cfg.MaxProbes == 0 {
		cfg.MaxProbes = 2
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 60 * time.Second
	}

	cb := &CircuitBreaker{
		config: cfg,
	}
	cb.setState(StateClosed)
	atomic.StoreInt64(&cb.currentBackoffNs, int64(cfg.Timeout))
	return cb
}

// State returns the current State of the circuit breaker.
func (cb *CircuitBreaker) State() State {
	return State(atomic.LoadUint32(&cb.state))
}

func (cb *CircuitBreaker) setState(s State) {
	atomic.StoreUint32(&cb.state, uint32(s))
}

// Allow checks if a request is permitted to proceed through the circuit breaker.
// If the circuit is OPEN, it evaluates whether the timeout has expired to transition to HALF_OPEN.
func (cb *CircuitBreaker) Allow() error {
	st := cb.State()

	switch st {
	case StateClosed:
		return nil

	case StateOpen:
		now := time.Now().UnixNano()
		nextAttempt := atomic.LoadInt64(&cb.nextAttemptNano)
		if now < nextAttempt {
			return ErrCircuitOpen
		}

		// Try transitioning from OPEN to HALF_OPEN
		if atomic.CompareAndSwapUint32(&cb.state, uint32(StateOpen), uint32(StateHalfOpen)) {
			// Reset successes and initialize probe count for half-open probing
			atomic.StoreUint64(&cb.successes, 0)
			atomic.StoreUint64(&cb.probeCount, 1)
			return nil
		}
		// If CAS failed, re-evaluate state
		return cb.Allow()

	case StateHalfOpen:
		// Enforce max active probes limit
		for {
			probes := atomic.LoadUint64(&cb.probeCount)
			if probes >= cb.config.MaxProbes {
				return ErrMaxProbesExited
			}
			if atomic.CompareAndSwapUint64(&cb.probeCount, probes, probes+1) {
				return nil
			}
		}

	default:
		return nil
	}
}

// Success records a successful execution, updating metrics and managing hysteresis transitions.
func (cb *CircuitBreaker) Success() {
	st := cb.State()

	switch st {
	case StateClosed:
		// Reset failure count on success when closed
		atomic.StoreUint64(&cb.failures, 0);

	case StateHalfOpen:
		succs := atomic.AddUint64(&cb.successes, 1)
		if succs >= cb.config.SuccessThreshold {
			// Transition from HALF_OPEN back to CLOSED
			if atomic.CompareAndSwapUint32(&cb.state, uint32(StateHalfOpen), uint32(StateClosed)) {
				atomic.StoreUint64(&cb.failures, 0)
				atomic.StoreUint64(&cb.successes, 0)
				atomic.StoreUint64(&cb.probeCount, 0)
				// Reset backoff to base timeout
				atomic.StoreInt64(&cb.currentBackoffNs, int64(cb.config.Timeout))
			}
		}
		// Decrement active probe count if tracked
		for {
			p := atomic.LoadUint64(&cb.probeCount)
			if p == 0 || atomic.CompareAndSwapUint64(&cb.probeCount, p, p-1) {
				break
			}
		}

	case StateOpen:
		// Ignored in Open state
	}
}

// Failure records a failed execution, triggering immediate open or escalation via exponential backoff.
func (cb *CircuitBreaker) Failure() {
	st := cb.State()

	switch st {
	case StateClosed:
	fails := atomic.AddUint64(&cb.failures, 1)
		if fails >= cb.config.FailureThreshold {
			if atomic.CompareAndSwapUint32(&cb.state, uint32(StateClosed), uint32(StateOpen)) {
				cb.tripOpen()
			}
		}

	case StateHalfOpen:
		// Any failure in half-open immediately trips back to OPEN with escalated backoff
		if atomic.CompareAndSwapUint32(&cb.state, uint32(StateHalfOpen), uint32(StateOpen)) {
			cb.tripOpen()
		}

	case StateOpen:
		// Already open
	}
}

func (cb *CircuitBreaker) tripOpen() {
	atomic.StoreUint64(&cb.successes, 0)
	atomic.StoreUint64(&cb.probeCount, 0)

	// Calculate exponential backoff
	current := atomic.LoadInt64(&cb.currentBackoffNs)
	next := time.Duration(float64(current) * cb.config.BackoffFactor)
	if next > cb.config.MaxBackoff {
		next = cb.config.MaxBackoff
	}
	atomic.StoreInt64(&cb.currentBackoffNs, int64(next))

	nextAttempt := time.Now().Add(next).UnixNano()
	atomic.StoreInt64(&cb.nextAttemptNano, nextAttempt)
}

// Reset manually resets the circuit breaker back to CLOSED state.
func (cb *CircuitBreaker) Reset() {
	atomic.StoreUint32(&cb.state, uint32(StateClosed))
	atomic.StoreUint64(&cb.failures, 0)
	atomic.StoreUint64(&cb.successes, 0)
	atomic.StoreUint64(&cb.probeCount, 0)
	atomic.StoreInt64(&cb.currentBackoffNs, int64(cb.config.Timeout))
	atomic.StoreInt64(&cb.nextAttemptNano, 0)
}
