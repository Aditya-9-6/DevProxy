package waf

import (
	"bytes"
	"sync"
)

// Matcher performs pattern scanning for WAF rule validation.
// It utilizes Go's standard library bytes.Contains which is highly optimized for performance.
type Matcher struct {
	mu       sync.RWMutex
	patterns [][]byte
}

// NewMatcher instantiates a Matcher with predefined attack signatures.
func NewMatcher(patterns []string) *Matcher {
	pats := make([][]byte, len(patterns))
	for i, p := range patterns {
		pats[i] = []byte(p)
	}
	return &Matcher{patterns: pats}
}

// Scan checks if any configured pattern exists within the input stream.
func (m *Matcher) Scan(data []byte) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, pat := range m.patterns {
		if bytes.Contains(data, pat) {
			return true
		}
	}
	return false
}

// AddPattern dynamically appends a new search pattern in a thread-safe manner.
func (m *Matcher) AddPattern(pattern string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.patterns = append(m.patterns, []byte(pattern))
}
