package waf

import (
	"bytes"
)

// NormalizePath performs zero-allocation URI normalization using sync.Pool buffers.
func NormalizePath(path string) string {
	buf := GetBuffer()
	defer PutBuffer(buf)

	// Percent decode first to handle encoded traversal patterns like %2e%2e/
	src := []byte(path)
	decoded := percentDecodeInto(buf[:0], src)

	// Normalize backslashes and slashes
	normalized := normalizeSlashes(decoded)

	// Resolve dot segments per RFC 3986
	resolved := resolvePathSegments(normalized)

	return string(resolved)
}

// IsDoubleEncoded checks if the path contains double-encoded characters.
func IsDoubleEncoded(path string) bool {
	for i := 0; i < len(path)-2; i++ {
		if path[i] == '%' && path[i+1] == '2' && (path[i+2] == '5' || path[i+2] == '3') {
			// Check for subsequent percent encoding
			for j := i + 3; j < len(path)-2; j++ {
				if path[j] == '%' {
					return true
				}
			}
		}
	}
	return false
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
	// Collapse multiple slashes
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
	// Preserve trailing slash if original had one and stack isn't empty
	if len(path) > 0 && path[len(path)-1] == '/' {
		buf.WriteByte('/')
	}
	return buf.Bytes()
}
