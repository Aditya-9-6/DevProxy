package waf

import (
	"bytes"
	"sort"
	"strings"
	"sync"
)

var (
	builderPool = sync.Pool{
		New: func() interface{} { return &strings.Builder{} },
	}
	slicePool = sync.Pool{
		New: func() interface{} { return make([]string, 0, 16) },
	}
)

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
