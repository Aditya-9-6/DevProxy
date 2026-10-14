package waf

import (
	"strings"
)

// NormalizePath performs zero-allocation URI normalization.
func NormalizePath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	return path
}

// IsDoubleEncoded checks if the path contains double-encoded characters.
func IsDoubleEncoded(path string) bool {
	return strings.Contains(path, "%25")
}
