package proxy

import (
	"sync/atomic"
)

// CacheLinePad64 prevents false sharing across CPU cores by padding atomic values.
type CacheLinePad64 [8]uint64

// ConnectionStateDescriptor holds cache-aligned atomic flags and counters for a demuxed TCP/WebSocket connection.
type ConnectionStateDescriptor struct {
	StateID          CacheLinePad64 // index 0 holds state bitmask/status
	BytesTransferred CacheLinePad64 // index 0 holds active transferred byte count
	ActiveFlag       CacheLinePad64 // index 0 holds 1 if active, 0 if inactive/pooled
	RemoteAddrHash   CacheLinePad64 // index 0 holds hashed remote endpoint identifier
}

// Reset clears descriptor state for reuse in the connection pool.
func (d *ConnectionStateDescriptor) Reset() {
	atomic.StoreUint64(&d.StateID[0], 0)
	atomic.StoreUint64(&d.BytesTransferred[0], 0)
	atomic.StoreUint64(&d.ActiveFlag[0], 0)
	atomic.StoreUint64(&d.RemoteAddrHash[0], 0)
}

// SetState updates the state ID atomically.
func (d *ConnectionStateDescriptor) SetState(state uint64) {
	atomic.StoreUint64(&d.StateID[0], state)
}

// GetState retrieves the state ID atomically.
func (d *ConnectionStateDescriptor) GetState() uint64 {
	return atomic.LoadUint64(&d.StateID[0])
}

// AddBytes atomically increments transmitted byte counters.
func (d *ConnectionStateDescriptor) AddBytes(n uint64) {
	atomic.AddUint64(&d.BytesTransferred[0], n)
}

// GetBytes retrieves transmitted byte count atomically.
func (d *ConnectionStateDescriptor) GetBytes() uint64 {
	return atomic.LoadUint64(&d.BytesTransferred[0])
}

// SetActive marks the descriptor active or inactive.
func (d *ConnectionStateDescriptor) SetActive(active bool) {
	var val uint64
	if active {
		val = 1
	}
	atomic.StoreUint64(&d.ActiveFlag[0], val)
}

// IsActive checks if the descriptor is currently active.
func (d *ConnectionStateDescriptor) IsActive() bool {
	return atomic.LoadUint64(&d.ActiveFlag[0]) == 1
}
