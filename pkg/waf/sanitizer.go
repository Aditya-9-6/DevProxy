package waf

import (
	"strings"
)

// SanitizeHeaderValue strips CR/LF characters to prevent HTTP Request Splitting attacks.
func SanitizeHeaderValue(val string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, val)
}
