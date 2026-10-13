package waf

// SIMDScanner provides low-level byte scanning primitives.
// In a production environment, this would utilize assembly or compiler intrinsics.
type SIMDScanner struct{}

func (s *SIMDScanner) ScanForCollision(data []byte, key []byte) int {
	count := 0
	for i := 0; i <= len(data)-len(key); i++ {
		if match(data[i:], key) {
			count++
		}
	}
	return count
}

func match(a, b []byte) bool {
	for i := range b {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
