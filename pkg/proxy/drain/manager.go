package drain

import (
	"sync"
	"sync/atomic"
)

// State constants for connection lifecycle.
const (
	StateActive int32 = iota
	StateDraining
	StateClosed
)

// Manager coordinates graceful connection shutdown using atomic state transitions.
type Manager struct {
	state atomic.Int32
	count atomic.Int64
	pool  sync.Pool
}

// NewManager creates a new drain manager.
func NewManager() *Manager {
	return &Manager{
		pool: sync.Pool{
			New: func() interface{} {
				return &ConnectionState{}
			},
		},
	}
}

// ConnectionState tracks individual connection lifecycle.
type ConnectionState struct {
	Active bool
}

// Register increments the active connection counter if not draining.
func (m *Manager) Register() bool {
	if m.state.Load() == StateDraining {
		return false
	}
	m.count.Add(1)
	return true
}

// Unregister decrements the active connection counter.
func (m *Manager) Unregister() {
	m.count.Add(-1)
}

// Drain initiates the shutdown process.
func (m *Manager) Drain() {
	m.state.Store(StateDraining)
}

// ActiveCount returns the number of currently tracked connections.
func (m *Manager) ActiveCount() int64 {
	return m.count.Load()
}
