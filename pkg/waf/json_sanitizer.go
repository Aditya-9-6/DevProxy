package waf

import (
	"bytes"
	"io"
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
// It uses an io.Writer to stream output directly, avoiding heap allocations.
func (s *Sanitizer) SanitizeStream(w io.Writer, input []byte, sensitiveFields [][]byte) error {
	// Use a simple state machine to track if we are inside a key
	// This is a simplified tokenizer for JSON key-value pairs
	for i := 0; i < len(input); i++ {
		found := false
		for _, field := range sensitiveFields {
			if bytes.HasPrefix(input[i:], field) {
				// Write key
				w.Write(field)
				i += len(field)

				// Skip whitespace and find colon
				for i < len(input) && input[i] != ':' {
					i++
				}
				w.Write([]byte(":"))

				// Redact value
				w.Write([]byte(" \"REDACTED\""))

				// Skip original value
				for i < len(input) && input[i] != ',' && input[i] != '}' {
					i++
				}
				i-- // Adjust for loop increment
				s.RedactionCount.Add(1)
				found = true
				break
			}
		}
		if !found {
			w.Write(input[i : i+1])
		}
	}
	return nil
}
