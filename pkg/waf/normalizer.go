package waf

import (
	"unicode"
)

// NormalizeHeaderKey converts header keys to canonical form and strips non-printable control characters.
// Returns a PooledBuffer that must be closed by the caller.
func NormalizeHeaderKey(key []byte) (*PooledBuffer, bool) {
	pbuf := GetPooledBuffer()
	if len(key) > len(pbuf.Data) {
		return nil, false
	}

	capitalize := true
	idx := 0
	for _, b := range key {
		if b < 32 || b == 127 {
			continue
		}
		r := rune(b)
		if capitalize {
			pbuf.Data[idx] = byte(unicode.ToUpper(r))
			capitalize = false
		} else {
			pbuf.Data[idx] = byte(unicode.ToLower(r))
		}
		if b == '-' {
			capitalize = true
		}
		idx++
	}
	pbuf.Data = pbuf.Data[:idx]
	return pbuf, true
}
