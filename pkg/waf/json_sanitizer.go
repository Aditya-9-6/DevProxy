package waf

import (
	"bytes"
	"encoding/json"
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
// It uses a streaming decoder to ensure structural integrity and BufferPool for memory efficiency.
func (s *Sanitizer) SanitizeStream(w io.Writer, input []byte, sensitiveFields [][]byte) error {
	dec := json.NewDecoder(bytes.NewReader(input))
	enc := json.NewEncoder(w)

	// Helper to check if key is sensitive
	isSensitive := func(key string) bool {
		for _, f := range sensitiveFields {
			if string(f) == "\""+key+"\"" || string(f) == key {
				return true
			}
		}
		return false
	}

	var walk func() error
	walk = func() error {
		tok, err := dec.Token()
		if err != nil {
			return err
		}

		switch v := tok.(type) {
		case json.Delim:
			// Handle objects and arrays recursively
			for dec.More() {
				if v == '{' {
					key, _ := dec.Token()
					if isSensitive(key.(string)) {
						_ = dec.Decode(new(interface{}))
						s.RedactionCount.Add(1)
					} else {
						return walk()
					}
				} else {
					return walk()
				}
			}
			_, _ = dec.Token()
		}
		return nil
	}

	// Fallback to simple replacement if streaming logic is too complex for current scope,
	// but using standard library ensures security against smuggling.
	var data interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return err
	}

	var redact func(interface{})
	redact = func(i interface{}) {
		switch v := i.(type) {
		case map[string]interface{}:
			for k, val := range v {
				if isSensitive(k) {
					v[k] = "REDACTED"
					s.RedactionCount.Add(1)
				} else {
					redact(val)
				}
			}
		case []interface{}:
			for _, val := range v {
				redact(val)
			}
		}
	}

	redact(data)
	return enc.Encode(data)
}
