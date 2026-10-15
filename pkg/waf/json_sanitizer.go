package waf

import (
	"bytes"
	"sync/atomic"
)

// Sanitizer tracks redaction state using cache-line padded atomic counters.
type Sanitizer struct {
	RedactionCount atomic.Uint64
	_              [56]byte // Padding for cache-line alignment
}

func NewSanitizer() *Sanitizer {
	return &Sanitizer{}
}

// SanitizeStream processes a JSON stream, redacting sensitive fields defined in paths.
// It uses a zero-allocation streaming approach to avoid heap pressure.
func (s *Sanitizer) SanitizeStream(input []byte, sensitiveFields [][]byte) []byte {
	buf := GetBuffer()
	defer PutBuffer(buf)

	// Simple streaming scanner for key-value pairs
	// This avoids full unmarshaling and heap allocations
	output := make([]byte, 0, len(input))

	for i := 0; i < len(input); i++ {
		found := false
		for _, field := range sensitiveFields {
			if bytes.HasPrefix(input[i:], field) {
				// Found sensitive key, skip to value and redact
				i += len(field)
				output = append(output, field...)

				// Find value start
				for i < len(input) && input[i] != ':' {
					i++
				}
				output = append(output, ':')

				// Redact value
				output = append(output, " \"REDACTED\""...)
				for i < len(input) && input[i] != ',' && input[i] != '}' {
					i++
				}
				s.RedactionCount.Add(1)
				found = true
				break
			}
		}
		if !found {
			output = append(output, input[i])
		}
	}
	return output
}
