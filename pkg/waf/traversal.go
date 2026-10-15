package waf

import (
	"strings"
)

// DetectTraversal inspects a path for directory traversal sequences with robust decoding and normalization.
func DetectTraversal(path string) bool {
	// Check raw or encoded traversal markers before/during normalization
	lower := strings.ToLower(path)
	if strings.Contains(lower, "%2e%2e") || strings.Contains(lower, "..%2f") || strings.Contains(lower, "%2f..") || strings.Contains(lower, "../") || strings.HasSuffix(path, "..") {
		return true
	}

	buf := GetBuffer()
	defer PutBuffer(buf)

	src := []byte(path)
	decoded := percentDecodeInto(buf[:0], src)
	normalized := normalizeSlashes(decoded)

	// Check resolved traversal attempt or explicit dot-dot sequences
	strNorm := string(normalized)
	if strings.Contains(strNorm, "../") || strings.HasSuffix(strNorm, "..") || strings.Contains(strNorm, "/../") {
		return true
	}

	return false
}
