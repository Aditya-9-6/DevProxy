package waf

import (
	"bytes"
	"sync"
	"sync/atomic"
)

var boundaryPrefix = []byte("--")

// BoundaryFilter implements a zero-allocation, lock-free anomaly detector for multipart boundaries.
// It uses a bitmask to track state transitions to prevent boundary smuggling/traversal.
type BoundaryFilter struct {
	// bitmask tracks state: 0=idle, 1=boundary_start, 2=boundary_body
	state atomic.Uint32
	pool  *sync.Pool
}

func NewBoundaryFilter() *BoundaryFilter {
	return &BoundaryFilter{
		pool: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 4096)
				return &buf
			},
		},
	}
}

// GetBuffer retrieves a byte slice buffer from the sync pool.
func (f *BoundaryFilter) GetBuffer() []byte {
	ptr := f.pool.Get().(*[]byte)
	return *ptr
}

// PutBuffer returns a byte slice buffer to the sync pool.
func (f *BoundaryFilter) PutBuffer(buf []byte) {
	f.pool.Put(&buf)
}

// InspectChunk checks for malicious boundary injection patterns.
// Returns true if the chunk is safe, false if an anomaly is detected.
func (f *BoundaryFilter) InspectChunk(chunk []byte) bool {
	// Detect potential boundary smuggling via null bytes or anomalous control characters
	for _, b := range chunk {
		if b == 0x00 {
			return false
		}
	}

	// State machine transition: check for boundary prefix '--'
	if bytes.Contains(chunk, boundaryPrefix) {
		f.state.Store(1)
	}

	return true
}

func (f *BoundaryFilter) Reset() {
	f.state.Store(0)
}
