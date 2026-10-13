package waf

import (
	"unicode"
)

// NormalizeHeaderKey converts header keys to canonical form and strips non-printable control characters.
// Returns a slice pointing into the pooled buffer. The caller must return the buffer to the pool using PutBuffer(buf).
func NormalizeHeaderKey(key []byte) []byte {
	buf := GetBuffer()
	if len(key) > len(buf) {
		return nil
	}

	capitalize := true
	idx := 0
	for _, b := range key {
		if b < 32 || b == 127 {
			continue
		}
		r := rune(b)
		if capitalize {
			buf[idx] = byte(unicode.ToUpper(r))
			capitalize = false
		} else {
			buf[idx] = byte(unicode.ToLower(r))
		}
		if b == '-' {
			capitalize = true
		}
		idx++
	}
	return buf[:idx]
}
