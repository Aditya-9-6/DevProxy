package waf

import (
	"bytes"
)

// Matcher performs vectorized SIMD-accelerated pattern scanning for WAF rule validation.
type Matcher struct {
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

// Scan checks if any configured pattern exists within the input stream using optimized byte searching.
func (m *Matcher) Scan(data []byte) bool {
	for _, pat := range m.patterns {
		if bytes.Contains(data, pat) {
			return true
		}
	}
	return false
}

// AddPattern dynamically appends a new search pattern.
func (m *Matcher) AddPattern(pattern string) {
	m.patterns = append(m.patterns, []byte(pattern))
}
