package frame

// ValidateUTF8 checks if the provided byte slice is valid UTF-8 according to RFC 3629 / RFC 6455.
func ValidateUTF8(p []byte) bool {
	n := len(p)
	i := 0
	for i < n {
		p0 := p[i]
		if p0 < 0x80 {
			i++
			continue
		}
		if (p0 & 0xE0) == 0xC0 {
			if i+1 >= n {
				return false
			}
			p1 := p[i+1]
			if (p1 & 0xC0) != 0x80 {
				return false
			}
			if p0 < 0xC2 {
				return false // Overlong encoding
			}
			i += 2
			continue
		}
		if (p0 & 0xF0) == 0xE0 {
			if i+2 >= n {
				return false
			}
			p1 := p[i+1]
			p2 := p[i+2]
			if (p1 & 0xC0) != 0x80 || (p2 & 0xC0) != 0x80 {
				return false
			}
			if p0 == 0xE0 && (p1 & 0xE0) == 0x80 {
				return false // Overlong encoding
			}
			if p0 == 0xED && (p1 & 0xE0) == 0xA0 {
				return false // Surrogate halves
			}
			i += 3
			continue
		}
		if (p0 & 0xF8) == 0xF0 {
			if i+3 >= n {
				return false
			}
			p1 := p[i+1]
			p2 := p[i+2]
			p3 := p[i+3]
			if (p1 & 0xC0) != 0x80 || (p2 & 0xC0) != 0x80 || (p3 & 0xC0) != 0x80 {
				return false
			}
			if p0 == 0xF0 && (p1 & 0xF0) == 0x80 {
				return false // Overlong encoding
			}
			if p0 > 0xF4 || (p0 == 0xF4 && p1 > 0x8F) {
				return false // Out of Unicode range
			}
			i += 4
			continue
		}
		return false
	}
	return true
}
