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

	// Handle ClientHellos larger than the default 4096 buffer
	if n >= 5 {
		recordLen := int(binary.BigEndian.Uint16(buf[3:5])) + 5
		if recordLen > len(buf) {
			largeBuf := make([]byte, recordLen)
			copy(largeBuf, buf[:n])
			_, err = io.ReadFull(reader, largeBuf[n:])
			if err != nil && err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
				return "", err
			}
			buf = largeBuf
			n = recordLen
		}
	}

	return parseClientHello(buf[:n])
}

func parseClientHello(buf []byte) (string, error) {
	n := len(buf)
	if n < 5 {
		return "", errors.New("insufficient data for TLS header")
	}

	if buf[0] != 0x16 {
		return "", errors.New("not a TLS handshake")
	}

	if n < 9 || buf[5] != 0x01 {
		return "", errors.New("not a ClientHello")
	}

	pos, err := skipClientHelloFixedFields(buf, n)
	if err != nil {
		return "", err
	}

	pos, err = skipCipherSuites(buf, pos, n)
	if err != nil {
		return "", err
	}

	pos, err = skipCompressionMethods(buf, pos, n)
	if err != nil {
		return "", err
	}

	return parseExtensions(buf, pos, n)
}

func skipClientHelloFixedFields(buf []byte, n int) (int, error) {
	pos := 43
	if pos+1 > n {
		return 0, errors.New("truncated ClientHello")
	}
	sessionIDLen := int(buf[pos])
	pos += 1 + sessionIDLen
	return pos, nil
}

func skipCipherSuites(buf []byte, pos, n int) (int, error) {
	if pos+2 > n {
		return 0, errors.New("truncated cipher suites")
	}
	cipherLen := int(binary.BigEndian.Uint16(buf[pos : pos+2]))
	pos += 2 + cipherLen
	return pos, nil
}

func skipCompressionMethods(buf []byte, pos, n int) (int, error) {
	if pos+1 > n {
		return 0, errors.New("truncated compression methods")
	}
	compLen := int(buf[pos])
	pos += 1 + compLen
	return pos, nil
}

func parseExtensions(buf []byte, pos, n int) (string, error) {
	if pos+2 > n {
		return "", nil // No extensions
	}
	extLen := int(binary.BigEndian.Uint16(buf[pos : pos+2]))
	pos += 2

	limit := pos + extLen
	if limit > n {
		limit = n
	}

	for pos+4 <= limit && pos+4 <= n {
		extType := binary.BigEndian.Uint16(buf[pos : pos+2])
		extDataLen := int(binary.BigEndian.Uint16(buf[pos+2 : pos+4]))
		pos += 4

		if extType == 0x0000 {
			if pos+5 <= n && pos+5 <= limit {
				nameLen := int(binary.BigEndian.Uint16(buf[pos+3 : pos+5]))
				if pos+5+nameLen <= n && pos+5+nameLen <= limit {
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
