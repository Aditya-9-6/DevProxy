package waf

// SanitizeHeaderValue strips CR/LF characters to prevent HTTP Request Splitting attacks.
// Operates on []byte to avoid allocations.
func SanitizeHeaderValue(val []byte) []byte {
	buf := GetBuffer()
	defer PutBuffer(buf)

	idx := 0
	for _, b := range val {
		if b == '\r' || b == '\n' {
			continue
		}
		buf[idx] = b
		idx++
	}
	result := make([]byte, idx)
	copy(result, buf[:idx])
	return result
}
