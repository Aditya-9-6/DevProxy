package security

import (
	"net/http"
	"sync/atomic"
)

// CacheLinePad prevents false sharing on hot atomic counters.
type CacheLinePad [8]uint64

// SmugglingSanitizer tracks potential HTTP smuggling attempts and uses CacheLinePad
// to ensure the hot atomic counter sits on its own cache line.
type SmugglingSanitizer struct {
	_pad0              CacheLinePad
	ContradictionCount uint64
	_pad1              CacheLinePad
}

func NewSmugglingSanitizer() *SmugglingSanitizer {
	return &SmugglingSanitizer{}
}

// asciiLower returns the lowercase equivalent of a byte without allocations.
func asciiLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}

// asciiEqualFold performs a zero-allocation case-insensitive string comparison.
func asciiEqualFold(s, target string) bool {
	if len(s) != len(target) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if asciiLower(s[i]) != asciiLower(target[i]) {
			return false
		}
	}
	return true
}

// containsChunkedZeroAlloc checks if a value contains 'chunked' case-insensitively without allocations.
func containsChunkedZeroAlloc(s string) bool {
	target := "chunked"
	targetLen := len(target)
	if len(s) < targetLen {
		return false
	}
	for i := 0; i <= len(s)-targetLen; i++ {
		match := true
		for j := 0; j < targetLen; j++ {
			if asciiLower(s[i+j]) != target[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Sanitize checks for dual Content-Length and Transfer-Encoding headers and smuggling variations.
func (s *SmugglingSanitizer) Sanitize(headers http.Header) bool {
	hasCL := false
	hasTE := false

	for k, values := range headers {
		// Check header keys case-insensitively
		isCL := asciiEqualFold(k, "Content-Length")
		isTE := asciiEqualFold(k, "Transfer-Encoding")

		if isCL && len(values) > 0 {
			// Multiple Content-Length headers or non-empty values
			hasCL = true
		}

		if isTE && len(values) > 0 {
			for _, val := range values {
				if val != "" {
					hasTE = true
					if !containsChunkedZeroAlloc(val) {
						return false
					}
				}
			}
		}
	}

	if hasCL && hasTE {
		atomic.AddUint64(&s.ContradictionCount, 1)
		return false
	}

	return true
}
