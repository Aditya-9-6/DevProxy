package waf

import "bytes"

// ByteScanner provides efficient byte scanning primitives.
type ByteScanner struct{}

// ScanForCollision counts occurrences of key in data using standard library optimization.
func (s *ByteScanner) ScanForCollision(data []byte, key []byte) int {
	if len(key) == 0 {
		return 0
	}
	return bytes.Count(data, key)
}
