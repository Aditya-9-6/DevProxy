package waf

import (
	"bytes"
	"sort"
	"strings"
	"sync"
)

var builderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// NormalizeContentDisposition strips malicious characters from headers.
func NormalizeContentDisposition(header []byte) []byte {
	return bytes.ReplaceAll(bytes.ReplaceAll(header, []byte("\r"), nil), []byte("\n"), nil)
}

// NormalizeQueryString sorts parameters to prevent bypasses without allocations.
func NormalizeQueryString(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	parts := strings.Split(rawQuery, "&")
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
	return res
}
