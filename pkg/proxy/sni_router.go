package proxy

import (
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"sync"
)

var sniBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

// SNIRouter extracts the SNI from a TLS ClientHello without decrypting.
type SNIRouter struct{}

func NewSNIRouter() *SNIRouter {
	return &SNIRouter{}
}

// ExtractSNI parses the TLS ClientHello to find the SNI extension.
func (r *SNIRouter) ExtractSNI(reader io.Reader) (string, error) {
	buf := sniBufferPool.Get().([]byte)
	defer sniBufferPool.Put(buf)

	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	if n < 5 {
		return "", errors.New("insufficient data for TLS header")
	}

	// TLS Record Header: Type (1), Version (2), Length (2)
	if buf[0] != 0x16 { // Handshake
		return "", errors.New("not a TLS handshake")
	}

	// Handshake Type (1), Length (3)
	if n < 9 || buf[5] != 0x01 { // ClientHello
		return "", errors.New("not a ClientHello")
	}

	// Skip fixed fields: Version (2), Random (32), SessionID Length (1)
	pos := 43
	if pos+1 > n {
		return "", errors.New("truncated ClientHello")
	}
	sessionIDLen := int(buf[pos])
	pos += 1 + sessionIDLen

	// Cipher Suites
	if pos+2 > n {
		return "", errors.New("truncated cipher suites")
	}
	cipherLen := int(binary.BigEndian.Uint16(buf[pos : pos+2]))
	pos += 2 + cipherLen

	// Compression Methods
	if pos+1 > n {
		return "", errors.New("truncated compression methods")
	}
	compLen := int(buf[pos])
	pos += 1 + compLen

	// Extensions
	if pos+2 > n {
		return "", nil // No extensions
	}
	extLen := int(binary.BigEndian.Uint16(buf[pos : pos+2]))
	pos += 2

	limit := pos + extLen
	for pos+4 <= limit && pos+4 <= n {
		extType := binary.BigEndian.Uint16(buf[pos : pos+2])
		extDataLen := int(binary.BigEndian.Uint16(buf[pos+2 : pos+4]))
		pos += 4

		if extType == 0x0000 { // SNI
			if pos+5 <= n && pos+5 <= limit {
				// listLen := int(binary.BigEndian.Uint16(buf[pos : pos+2]))
				sniType := buf[pos+2]
				nameLen := int(binary.BigEndian.Uint16(buf[pos+3 : pos+5]))
				if pos+5+nameLen <= n {
					_ = sniType
					return string(buf[pos+5 : pos+5+nameLen]), nil
				}
			}
		}
		pos += extDataLen
	}

	return "", errors.New("SNI not found")
}

// MatchWildcard checks if a domain matches a pattern (e.g., *.example.com).
func MatchWildcard(pattern, domain string) bool {
	if pattern == domain {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:]
		return strings.HasSuffix(domain, suffix)
	}
	return false
}
