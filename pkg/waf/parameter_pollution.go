package waf

import (
	"net/url"
	"sync/atomic"
)

// ParameterValidator tracks parameter multiplicity using atomic counters.
type ParameterValidator struct {
	// Multiplicity counters for common parameter names (e.g., 'id', 'user')
	// Using a fixed array for O(1) access without map allocations.
	Counters [256]CacheLinePad
}

// NewParameterValidator initializes a new validator.
func NewParameterValidator() *ParameterValidator {
	return &ParameterValidator{}
}

// ValidateQuery inspects URL parameters for duplicate keys (pollution).
// Returns true if the request is clean, false if pollution is detected.
func (v *ParameterValidator) ValidateQuery(values url.Values) bool {
	for key := range values {
		if len(values[key]) > 1 {
			return false
		}
	}
	return true
}

// RecordParamUsage increments the usage count for a parameter name.
func (v *ParameterValidator) RecordParamUsage(paramName string) {
	idx := hashParam(paramName)
	atomic.AddUint64(&v.Counters[idx][0], 1)
}

func hashParam(s string) uint8 {
	var h uint8
	for i := 0; i < len(s); i++ {
		h = h*31 + s[i]
	}
	return h
}
