package waf

import (
	"strings"
	"unicode"
)

// NormalizeHeaderKey converts header keys to canonical form (e.g., "content-type" -> "Content-Type")
// and strips non-printable control characters to prevent request smuggling.
func NormalizeHeaderKey(key string) string {
	var b strings.Builder
	capitalize := true
	for _, r := range key {
		if r < 32 || r == 127 {
			continue
		}
		if capitalize {
			b.WriteRune(unicode.ToUpper(r))
			capitalize = false
		} else {
			b.WriteRune(unicode.ToLower(r))
		}
		if r == '-' {
			capitalize = true
		}
	}
	return b.String()
}
