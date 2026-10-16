package waf

// SanitizeHeaderValue strips CR/LF characters to prevent HTTP Request Splitting attacks.
// Returns a PooledBuffer that must be closed by the caller.
func SanitizeHeaderValue(val []byte) (*PooledBuffer, bool) {
	pbuf := GetPooledBuffer()
	if len(val) > len(pbuf.Data) {
		return nil, false
	}

	idx := 0
	for _, b := range val {
		if b == '\r' || b == '\n' {
			continue
		}
		pbuf.Data[idx] = b
		idx++
	}
	pbuf.Data = pbuf.Data[:idx]
	return pbuf, true
}
