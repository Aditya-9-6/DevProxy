package waf

import (
	"bytes"
	"strings"
)

// Normalizer handles lock-free, zero-allocation HTTP header key/value normalization.
type Normalizer struct{}

// NewNormalizer creates a new Header Normalizer instance.
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// NormalizeKey normalizes header keys to canonical Pascal-Case (e.g. "content-type" -> "Content-Type").
// Operates with zero allocations using pre-allocated scratch buffers when possible.
func (n *Normalizer) NormalizeKey(key []byte, scratch []byte) []byte {
	if len(key) == 0 {
		return nil
	}

	if cap(scratch) < len(key) {
		scratch = make([]byte, len(key))
	} else {
		scratch = scratch[:len(key)]
	}

	capitolNext := true
	for i := 0; i < len(key); i++ {
		c := key[i]
		if capitolNext && c >= 'a' && c <= 'z' {
			scratch[i] = c - 32
		} else if !capitolNext && c >= 'A' && c <= 'Z' {
			scratch[i] = c + 32
		} else {
			scratch[i] = c
		}

		if c == '-' {
			capitolNext = true
		} else {
			capitolNext = false
		}
	}
	return scratch
}

// NormalizeString provides a string wrapper for NormalizeKey.
func (n *Normalizer) NormalizeString(key string) string {
	bufPtr := GetBuffer()
	defer PutBuffer(bufPtr)
	raw := []byte(key)
	norm := n.NormalizeKey(raw, *bufPtr)
	return string(norm)
}

// NormalizeHeaderMap normalizes a map of HTTP headers.
func (n *Normalizer) NormalizeHeaderMap(headers map[string][]string) map[string][]string {
	out := make(map[string][]string, len(headers))
	bufPtr := GetBuffer()
	defer PutBuffer(bufPtr)

	for k, v := range headers {
		normKey := string(n.NormalizeKey([]byte(k), *bufPtr))
		out[normKey] = v
	}
	return out
}

// FastContains checks if a header value contains forbidden request splitting characters (\r, \n).
func FastContainsCRLF(val []byte) bool {
	return bytes.IndexAny(val, "\r\n") >= 0
}

// CanonicalizeHeaderName ensures standard casing for known headers.
func CanonicalizeHeaderName(name string) string {
	switch strings.ToLower(name) {
	case "content-type":
		return "Content-Type"
	case "content-length":
		return "Content-Length"
	case "authorization":
		return "Authorization"
	case "user-agent":
		return "User-Agent"
	case "host":
		return "Host"
	default:
		return name
	}
}
