package waf

import (
	"hash/fnv"
	"strings"
	"sync/atomic"
)

// ParameterValidator tracks parameter multiplicity using atomic counters.
type ParameterValidator struct {
	// Multiplicity counters for common parameter names.
	// Using a fixed array for O(1) access without map allocations.
	Counters [256]CacheLinePad
}

// NewParameterValidator initializes a new validator.
func NewParameterValidator() *ParameterValidator {
	return &ParameterValidator{}
}

// ValidateQuery inspects raw query strings for duplicate keys (pollution).
// This avoids net/url.Values heap allocations.
func (v *ParameterValidator) ValidateQuery(query string) bool {
	if query == "" {
		return true
	}
	// Use a map for tracking seen keys. While this allocates, it is necessary for arbitrary key sets.
	// For zero-allocation, a bloom filter or fixed-size hash table would be required.
	seen := make(map[string]struct{}, 8)
	parts := strings.Split(query, "&")
	for _, part := range parts {
		key := part
		if idx := strings.Index(part, "="); idx != -1 {
			key = part[:idx]
		}
		key = strings.ToLower(key)
		if _, exists := seen[key]; exists {
			return false
		}
		seen[key] = struct{}{}
	}
	return true
}

// RecordParamUsage increments the usage count for a parameter name using FNV-1a.
func (v *ParameterValidator) RecordParamUsage(paramName string) {
	idx := hashParam(paramName)
	atomic.AddUint64(&v.Counters[idx][0], 1)
}

func hashParam(s string) uint8 {
	h := fnv.New32a()
	h.Write([]byte(strings.ToLower(s)))
	return uint8(h.Sum32() % 256)
}
