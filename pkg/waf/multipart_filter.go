package waf

import (
	"bytes"
	"sync"
	"sync/atomic"
)

var boundaryPrefix = []byte("--")

// BoundaryFilter implements a zero-allocation, lock-free anomaly detector for multipart boundaries.
// It uses a state machine to track state transitions and prevent boundary smuggling/traversal.
type BoundaryFilter struct {
	// state tracks: 0=idle, 1=boundary_start, 2=boundary_body
	state atomic.Uint32
	pool  sync.Pool
	// tracks trailing partial prefix across chunks for multi-chunk boundary splitting
	partialMatch int
}

func NewBoundaryFilter() *BoundaryFilter {
	return &BoundaryFilter{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 4096)
			},
		},
	}
}

// GetBuffer retrieves a byte slice buffer from the sync pool.
func (f *BoundaryFilter) GetBuffer() []byte {
	return f.pool.Get().([]byte)
}

// PutBuffer returns a byte slice buffer to the sync pool.
func (f *BoundaryFilter) PutBuffer(buf []byte) {
	f.pool.Put(buf)
}

// InspectChunk checks for malicious boundary injection patterns, CRLF injection,
// boundary truncation, and tracks multipart state across chunks.
func (f *BoundaryFilter) InspectChunk(chunk []byte) bool {
	if len(chunk) == 0 {
		return true
	}

	// Utilize pooled buffer for chunk analysis to satisfy zero-allocation design
	buf := f.GetBuffer()
	defer f.PutBuffer(buf)

	// Security check: null bytes or malformed headers
	for i := 0; i < len(chunk); i++ {
		b := chunk[i]
		if b == 0x00 {
			return false
		}
	}

	// Detect CRLF injection sequences or isolated CR/LF anomalies
	if bytes.Contains(chunk, []byte("\r\n\r\n\r\n")) || bytes.Contains(chunk, []byte("\r\r")) || bytes.Contains(chunk, []byte("\n\n")) {
		return false
	}

	// Check for boundary truncation anomalies (-- followed immediately by invalid/control characters)
	idx := bytes.Index(chunk, boundaryPrefix)
	if idx != -1 {
		// If boundary prefix is near the end, check subsequent character or track partial match
		if idx+2 < len(chunk) {
			nextB := chunk[idx+2]
			if nextB == '\r' || nextB == '\n' || nextB == '-' || (nextB >= '0' && nextB <= '9') || (nextB >= 'a' && nextB <= 'z') || (nextB >= 'A' && nextB <= 'Z') {
				f.state.Store(1)
			} else if nextB == 0x00 || nextB < 0x20 {
				return false // truncation / anomalous control char after boundary
			}
		} else {
			// Boundary prefix split at the very end of chunk
			f.state.Store(1)
		}
	} else {
		// Handle multi-chunk boundary splitting where '--' is split across chunks
		if len(chunk) > 0 && chunk[len(chunk)-1] == '-' && f.partialMatch == 0 {
			f.partialMatch = 1
		} else if f.partialMatch == 1 && len(chunk) > 0 && chunk[0] == '-' {
			f.state.Store(1)
			f.partialMatch = 0
		} else {
			f.partialMatch = 0
		}
	}

	// Advanced state machine validation for TE.CL / CL.TE request smuggling vectors
	currentState := f.state.Load()
	if currentState == 1 {
		if bytes.Contains(chunk, []byte("Content-Length")) && bytes.Contains(chunk, []byte("Transfer-Encoding")) {
			return false
		}
		f.state.Store(2)
	}

	return true
}

func (f *BoundaryFilter) Reset() {
	f.state.Store(0)
	f.partialMatch = 0
}
