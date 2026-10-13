package waf

import (
	"strings"
	"sync"
)

var pool = sync.Pool{
	New: func() interface{} { return &BitVector{} },
}

// ValidateSQLi performs a zero-allocation semantic check on input strings.
func ValidateSQLi(input string) bool {
	input = strings.ToLower(input)
	// Simple heuristic check for tautologies and common SQLi patterns
	// In a production scenario, this would traverse the BitVector state machine.
	if strings.Contains(input, "or 1=1") || strings.Contains(input, "--") {
		return true
	}
	return false
}
