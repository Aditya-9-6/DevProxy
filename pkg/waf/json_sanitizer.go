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

	sensitiveMap := buildSensitiveMap(sensitiveFields)
	s.redactValue(data, 0, sensitiveMap)
	return json.NewEncoder(w).Encode(data)
}

func buildSensitiveMap(sensitiveFields [][]byte) map[string]struct{} {
	sensitiveMap := make(map[string]struct{}, len(sensitiveFields))
	for _, f := range sensitiveFields {
		sensitiveMap[string(f)] = struct{}{}
	}
	return sensitiveMap
}

func (s *Sanitizer) redactValue(i interface{}, depth int, sensitiveMap map[string]struct{}) {
	if depth > maxDepth {
		return
	}

	switch v := i.(type) {
	case map[string]interface{}:
		s.redactMap(v, depth, sensitiveMap)
	case []interface{}:
		s.redactSlice(v, depth, sensitiveMap)
	}
}

func (s *Sanitizer) redactMap(v map[string]interface{}, depth int, sensitiveMap map[string]struct{}) {
	for k, val := range v {
		if _, ok := sensitiveMap[k]; ok {
			v[k] = "REDACTED"
			s.RedactionCount.Add(1)
		} else {
			s.redactValue(val, depth+1, sensitiveMap)
		}
	}
}

func (s *Sanitizer) redactSlice(v []interface{}, depth int, sensitiveMap map[string]struct{}) {
	for _, val := range v {
		s.redactValue(val, depth+1, sensitiveMap)
	}
}
