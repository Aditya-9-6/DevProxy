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

// ConnectionState tracks individual connection lifecycle and is pooled for zero-alloc usage.
type ConnectionState struct {
	Active bool
}

// Manager coordinates graceful connection shutdown using atomic state transitions, zero-alloc pooling, and signal notifications.
type Manager struct {
	state atomic.Int32
	count atomic.Int64
	pool  sync.Pool
	mu    sync.Mutex
	cond  *sync.Cond
}

// NewManager initializes the drain manager.
func NewManager() *Manager {
	m := &Manager{
		pool: sync.Pool{
			New: func() interface{} {
				return &ConnectionState{Active: true}
			},
		},
	}
	m.cond = sync.NewCond(&m.mu)
	return m
}

// GetState acquires a pooled ConnectionState.
func (m *Manager) GetState() *ConnectionState {
	cs := m.pool.Get().(*ConnectionState)
	cs.Active = true
	return cs
}

// PutState returns a ConnectionState to the pool.
func (m *Manager) PutState(cs *ConnectionState) {
	cs.Active = false
	m.pool.Put(cs)
}

// Register increments the active connection counter if not draining.
func (m *Manager) Register() bool {
	if m.state.Load() == StateDraining {
		return false
	}
	m.count.Add(1)
	return true
}

// Unregister decrements the active connection counter and signals waiters if zero.
func (m *Manager) Unregister() {
	val := m.count.Add(-1)
	if val <= 0 {
		m.mu.Lock()
		m.cond.Broadcast()
		m.mu.Unlock()
	}
}

// Drain initiates the shutdown process.
func (m *Manager) Drain() {
	m.state.Store(StateDraining)
	if m.ActiveCount() == 0 {
		m.mu.Lock()
		m.cond.Broadcast()
		m.mu.Unlock()
	}
}

// ActiveCount returns the number of currently tracked connections.
func (m *Manager) ActiveCount() int64 {
	return m.count.Load()
}

// Wait blocks until the active connection count reaches zero.
func (m *Manager) Wait() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for m.ActiveCount() > 0 {
		m.cond.Wait()
	}
}
