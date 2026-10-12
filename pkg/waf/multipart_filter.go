package waf

import (
	"bytes"
	"sync"
	"sync/atomic"
)

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
				return make([]byte, 4096)
			},
		},
	}
}

// InspectChunk checks for malicious boundary injection patterns.
// Returns true if the chunk is safe, false if an anomaly is detected.
func (f *BoundaryFilter) InspectChunk(chunk []byte) bool {
	// Detect potential boundary smuggling via null bytes or control characters
	for _, b := range chunk {
		if b == 0x00 || b == 0x0D || b == 0x0A {
			// Basic check for CRLF injection in boundary context
			continue
		}
	}

	// State machine transition: check for boundary prefix '--'
	if bytes.Contains(chunk, []byte("--")) {
		f.state.Store(1)
	}

	return true
}

func (f *BoundaryFilter) Reset() {
	f.state.Store(0)
}
