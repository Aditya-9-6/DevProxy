package waf

import (
	"bytes"
	"sort"
	"strings"
	"sync"
	"unicode"
)

var (
	builderPool = sync.Pool{
		New: func() interface{} { return &strings.Builder{} },
	}
	slicePool = sync.Pool{
		New: func() interface{} { return make([]string, 0, 16) },
	}
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

// NormalizeContentDisposition strips malicious characters from headers.
func NormalizeContentDisposition(header []byte) []byte {
	return bytes.ReplaceAll(bytes.ReplaceAll(header, []byte("\r"), nil), []byte("\n"), nil)
}

// NormalizeQueryString sorts parameters to prevent bypasses without heap-allocating slices.
func NormalizeQueryString(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	parts := slicePool.Get().([]string)[:0]
	start := 0
	for i := 0; i <= len(rawQuery); i++ {
		if i == len(rawQuery) || rawQuery[i] == '&' {
			segment := rawQuery[start:i]
			if segment != "" {
				parts = append(parts, segment)
			}
			start = i + 1
		}
	}
	sort.Strings(parts)
	builder := builderPool.Get().(*strings.Builder)
	builder.Reset()
	for i, p := range parts {
		if i > 0 {
			builder.WriteByte('&')
		}
		builder.WriteString(p)
	}
	res := builder.String()
	builderPool.Put(builder)
	slicePool.Put(parts)
	return res
}