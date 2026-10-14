package waf

import (
	"strings"
)

// DetectTraversal inspects a path for directory traversal sequences.
func DetectTraversal(path string) bool {
	path = NormalizePath(path)
	if strings.Contains(path, "../") || strings.HasSuffix(path, "..") {
		return true
	}
	return false
}
