package waf

import (
	"encoding/json"
	"errors"
	"io"
	"sync/atomic"
)

const maxDepth = 64

// Sanitizer tracks redaction state using cache-line padded atomic counters.
type Sanitizer struct {
	RedactionCount atomic.Uint64
	_              [56]byte // Padding for cache-line alignment
}

func NewSanitizer() *Sanitizer {
	return &Sanitizer{}
}

// SanitizeStream processes a JSON input, redacting sensitive fields.
// It uses a depth-limited recursive approach to prevent stack exhaustion.
func (s *Sanitizer) SanitizeStream(w io.Writer, input []byte, sensitiveFields [][]byte) error {
	if len(input) == 0 {
		return errors.New("empty input")
	}

	var data interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return err
	}

	sensitiveMap := make(map[string]struct{}, len(sensitiveFields))
	for _, f := range sensitiveFields {
		sensitiveMap[string(f)] = struct{}{}
	}

	var redact func(interface{}, int)
	redact = func(i interface{}, depth int) {
		if depth > maxDepth {
			return
		}

		switch v := i.(type) {
		case map[string]interface{}:
			for k, val := range v {
				if _, ok := sensitiveMap[k]; ok {
					v[k] = "REDACTED"
					s.RedactionCount.Add(1)
				} else {
					redact(val, depth+1)
				}
			}
		case []interface{}:
			for _, val := range v {
				redact(val, depth+1)
			}
		}
	}

	redact(data, 0)
	return json.NewEncoder(w).Encode(data)
}
