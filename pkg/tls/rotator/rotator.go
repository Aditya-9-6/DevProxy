package rotator

import (
	"crypto/rand"
	"sync/atomic"
)

// STEK represents a Session Ticket Encryption Key.
type STEK [32]byte

// Rotator manages active and standby keys using atomic pointers for lock-free access.
type Rotator struct {
	active atomic.Pointer[STEK]
}

// NewRotator initializes a rotator with a random key.
func NewRotator() *Rotator {
	r := &Rotator{}
	r.Rotate()
	return r
}

// Get returns the current active STEK.
func (r *Rotator) Get() *STEK {
	return r.active.Load()
}

// Rotate generates a new key and updates the active pointer atomically.
func (r *Rotator) Rotate() {
	var newKey STEK
	_, _ = rand.Read(newKey[:])
	r.active.Store(&newKey)
}
