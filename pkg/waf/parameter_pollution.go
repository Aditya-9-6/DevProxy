package waf

import (
	"hash/fnv"
	"net/url"
	"strings"
	"sync/atomic"
)

// ParameterValidator tracks parameter multiplicity using atomic counters.
type ParameterValidator struct {
	Counters [256]CacheLinePad
}

// NewParameterValidator initializes a new validator.
func NewParameterValidator() *ParameterValidator {
	return &ParameterValidator{}
}

// ValidateQuery inspects raw query strings for duplicate keys (pollution).
// This avoids net/url.Values heap allocations by utilizing a stack-allocated bitset.
func (v *ParameterValidator) ValidateQuery(query string) bool {
	if query == "" {
		return true
	}

	// Increased bitset to 1024 bits (16 * 64) to reduce collision probability.
	var seenBits [16]uint64

	start := 0
	for i := 0; i <= len(query); i++ {
		if i == len(query) || query[i] == '&' || query[i] == ';' {
			if start < i {
				part := query[start:i]
				key := part
				if idx := strings.IndexByte(part, '='); idx != -1 {
					key = part[:idx]
				}

				if unescaped, err := url.QueryUnescape(key); err == nil {
					key = unescaped
				}

				key = strings.ToLower(key)
				hash := fnv32a(key)
				wordIdx := (hash >> 6) & 15
				bitIdx := hash & 63

				if (seenBits[wordIdx] & (1 << bitIdx)) != 0 {
					return false
				}
				seenBits[wordIdx] |= (1 << bitIdx)
			}
			start = i + 1
		}
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

func fnv32a(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
