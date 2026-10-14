package waf

import (
	"bytes"
)

// NormalizeContentDisposition strips malicious characters from headers.
func NormalizeContentDisposition(header []byte) []byte {
	// Remove potential CRLF injection sequences
	return bytes.ReplaceAll(bytes.ReplaceAll(header, []byte("\r"), nil), []byte("\n"), nil)
}
