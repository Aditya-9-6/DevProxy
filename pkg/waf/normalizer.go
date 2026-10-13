package waf

import (
	"unicode"
)

// NormalizeHeaderKey converts header keys to canonical form (e.g., "content-type" -> "Content-Type")
// and strips non-printable control characters to prevent request smuggling. Operates on []byte to avoid allocations.
func NormalizeHeaderKey(key []byte) []byte {
	buf := GetBuffer()
	defer PutBuffer(buf)

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
	result := make([]byte, idx)
	copy(result, buf[:idx])
	return result
}
