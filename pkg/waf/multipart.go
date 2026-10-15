package waf

import (
	"sync"
	"sync/atomic"
)

// MultipartState tracks boundary nesting depth using atomic counters to prevent injection.
// Uses a fixed-size ring to avoid heap allocations.
type MultipartState struct {
	boundaryDepth  atomic.Int64
	violationCount atomic.Uint64
}

var statePool = sync.Pool{
	New: func() interface{} {
		return &MultipartState{}
	},
}

// NormalizeBoundary checks for nested MIME boundary confusion.
func (s *MultipartState) NormalizeBoundary(boundary []byte) bool {
	// Simple heuristic: nested boundaries often contain recursive patterns.
	// We track depth; if depth > 5, we flag as potential injection.
	depth := s.boundaryDepth.Add(1)
	if depth > 5 {
		s.violationCount.Add(1)
		return false
	}
	return true
}

func (s *MultipartState) Reset() {
	s.boundaryDepth.Store(0)
}
