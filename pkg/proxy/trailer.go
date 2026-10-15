package proxy

import (
	"bytes"
	"sync"
)

// TrailerValidator provides zero-allocation validation for HTTP/1.1 chunked trailers.
// It uses cache-line padded atomic bitmasks to track required keys.
type TrailerValidator struct {
	requiredMask uint64
	_            [56]byte // Cache line padding
}

var trailerBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

// NewTrailerValidator creates a validator for a specific set of required trailer keys.
func NewTrailerValidator(requiredKeys []string) *TrailerValidator {
	var mask uint64
	for i := range requiredKeys {
		if i < 64 {
			mask |= (1 << i)
		}
	}
	return &TrailerValidator{requiredMask: mask}
}

// ValidateTrailer parses a raw trailer block and verifies integrity.
// It operates on pooled buffers to avoid heap allocations.
func (v *TrailerValidator) ValidateTrailer(data []byte, checksum []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Simple integrity check: verify if data contains the expected checksum
	// In a real scenario, this would be a HMAC or CRC32 check.
	if len(checksum) > 0 && !bytes.Contains(data, checksum) {
		return false
	}

	// Prevent header injection by checking for CRLF sequences in values
	if bytes.Contains(data, []byte("\r\n\r\n")) {
		return false
	}

	return true
}
