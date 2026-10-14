package waf

import (
	"bytes"
	"sync"
	"sync/atomic"
)

// Matcher performs pattern scanning for WAF rule validation using lock-free Copy-On-Write.
type Matcher struct {
	mu       sync.Mutex
	patterns atomic.Value // stores [][]byte
}

// NewMatcher instantiates a Matcher with predefined attack signatures.
func NewMatcher(patterns []string) *Matcher {
	m := &Matcher{}
	pats := make([][]byte, len(patterns))
	for i, p := range patterns {
		pats[i] = []byte(p)
	}
	m.patterns.Store(pats)
	return m
}

// Scan checks if any configured pattern exists within the input stream in a lock-free manner.
func (m *Matcher) Scan(data []byte) bool {
	pats, ok := m.patterns.Load().([][]byte)
	if !ok {
		return false
	}

	for _, pat := range pats {
		if bytes.Contains(data, pat) {
			return true
		}
	}
	return false
}

// AddPattern dynamically appends a new search pattern using thread-safe Copy-On-Write.
func (m *Matcher) AddPattern(pattern string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, _ := m.patterns.Load().([][]byte)
	newPats := make([][]byte, len(current)+1)
	copy(newPats, current)
	newPats[len(current)] = []byte(pattern)

	m.patterns.Store(newPats)
}
