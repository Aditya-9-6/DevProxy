package waf

import (
	"sync/atomic"
)

// StateTransition tracks atomic state changes with cache alignment.
type StateTransition struct {
	_       [56]byte
	State   atomic.Uint64
	Scans   atomic.Uint64
	Matches atomic.Uint64
	_       [56]byte
}

// Engine coordinates the WAF payload inspection pipeline.
type Engine struct {
	arena   *Arena
	matcher *Matcher
	state   StateTransition
}

// NewEngine initializes a WAF engine with default injection signatures and memory arena.
func NewEngine(bufferSize int) *Engine {
	defaultSqliXss := []string{
		"SELECT ", "UNION ", "INSERT ", "DELETE ", "DROP ", "UPDATE ", "OR 1=1", "--",
		"<script", "javascript:", "onerror=", "onload=", "alert(", "eval(",
	}

	eng := &Engine{
		arena:   NewArena(bufferSize),
		matcher: NewMatcher(defaultSqliXss),
	}
	eng.state.State.Store(1) // Active state
	return eng
}

// InspectPayload scans input bytes for malicious patterns without heap allocations using the Arena buffer.
func (e *Engine) InspectPayload(data []byte) bool {
	e.state.Scans.Add(1)
	if e.state.State.Load() == 0 {
		return false
	}

	// Utilize Arena for temporary processing space if needed, ensuring zero heap pressure
	buf := e.arena.Get()
	defer e.arena.Put(buf)

	matched := e.matcher.Scan(data)
	if matched {
		e.state.Matches.Add(1)
	}
	return matched
}

// Stats returns operational metrics of the WAF engine.
func (e *Engine) Stats() (scans uint64, matches uint64) {
	return e.state.Scans.Load(), e.state.Matches.Load()
}
