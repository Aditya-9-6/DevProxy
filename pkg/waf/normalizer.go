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

// NormalizePath performs zero-allocation URI normalization using sync.Pool buffers.
func NormalizePath(path string) string {
	buf := GetBuffer()
	defer PutBuffer(buf)

	src := []byte(path)
	decoded := percentDecodeInto(buf[:0], src)
	normalized := normalizeSlashes(decoded)
	resolved := resolvePathSegments(normalized)

	return string(resolved)
}

// IsDoubleEncoded checks if the path contains double-encoded characters.
func IsDoubleEncoded(path string) bool {
	for i := 0; i < len(path)-2; i++ {
		if path[i] == '%' && path[i+1] == '2' && (path[i+2] == '5' || path[i+2] == '3') {
			for j := i + 3; j < len(path)-2; j++ {
				if path[j] == '%' {
					return true
				}
			}
		}
	}
	return false
}

// NormalizeContentDisposition strips malicious characters from headers.
func NormalizeContentDisposition(header []byte) []byte {
	return bytes.ReplaceAll(bytes.ReplaceAll(header, []byte("\r"), nil), []byte("\n"), nil)
}

func percentDecodeInto(dst, src []byte) []byte {
	for i := 0; i < len(src); i++ {
		if src[i] == '%' && i+2 < len(src) {
			v1 := unhex(src[i+1])
			v2 := unhex(src[i+2])
			if v1 >= 0 && v2 >= 0 {
				dst = append(dst, byte(v1<<4|v2))
				i += 2
				continue
			}
		}
		dst = append(dst, src[i])
	}
	return dst
}

func unhex(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c - 'a' + 10)
	case 'A' <= c && c <= 'F':
		return int(c - 'A' + 10)
	}
	return -1
}

func normalizeSlashes(src []byte) []byte {
	for i, b := range src {
		if b == '\\' {
			src[i] = '/'
		}
	}
	out := src[:0]
	lastSlash := false
	for _, b := range src {
		if b == '/' {
			if !lastSlash {
				out = append(out, b)
				lastSlash = true
			}
		} else {
			out = append(out, b)
			lastSlash = false
		}
	}
	return out
}

func resolvePathSegments(path []byte) []byte {
	parts := bytes.Split(path, []byte("/"))
	var stack [][]byte
	for _, p := range parts {
		if len(p) == 0 || bytes.Equal(p, []byte(".")) {
			continue
		}
		if bytes.Equal(p, []byte("..")) {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		} else {
			stack = append(stack, p)
		}
	}

	if len(stack) == 0 {
		return []byte("/")
	}

	var buf bytes.Buffer
	for _, s := range stack {
		buf.WriteByte('/')
		buf.Write(s)
	}
	if len(path) > 0 && path[len(path)-1] == '/' {
		buf.WriteByte('/')
	}
	return buf.Bytes()
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
