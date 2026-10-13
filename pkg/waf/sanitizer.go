package waf

// SanitizeHeaderValue strips CR/LF characters to prevent HTTP Request Splitting attacks.
// Returns a slice pointing into the pooled buffer. The caller must return the buffer to the pool using PutBuffer(buf).
func SanitizeHeaderValue(val []byte) []byte {
	buf := GetBuffer()
	if len(val) > len(buf) {
		return nil
	}

	idx := 0
	for _, b := range val {
		if b == '\r' || b == '\n' {
			continue
		}
		buf[idx] = b
		idx++
	}
	return buf[:idx]
}
