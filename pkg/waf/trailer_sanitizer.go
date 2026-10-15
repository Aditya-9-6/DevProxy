package waf

import (
	"bytes"
	"sync"
	"sync/atomic"
)

// TrailerSanitizer inspects HTTP streams for smuggling and malformed chunks.
type TrailerSanitizer struct {
	AnomalyCount CacheLinePad
	bufferPool   sync.Pool
}

func NewTrailerSanitizer() *TrailerSanitizer {
	return &TrailerSanitizer{
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 32*1024)
			},
		},
	}
}

// SanitizeChunk inspects a chunk for smuggling patterns or invalid boundaries.
func (s *TrailerSanitizer) SanitizeChunk(chunk []byte) bool {
	// Check for trailer-header smuggling: CRLF followed by header-like patterns
	if bytes.Contains(chunk, []byte("\r\n\r\n")) || bytes.Contains(chunk, []byte("\r\nTransfer-Encoding:")) {
		atomic.AddUint64(&s.AnomalyCount[0], 1)
		return false
	}
	return true
}

// ValidateTrailer inspects trailer headers for illegal characters or smuggling.
func (s *TrailerSanitizer) ValidateTrailer(trailer []byte) bool {
	if bytes.ContainsAny(trailer, "\r\n") {
		atomic.AddUint64(&s.AnomalyCount[0], 1)
		return false
	}
	return true
}

func (s *TrailerSanitizer) GetAnomalyCount() uint64 {
	return atomic.LoadUint64(&s.AnomalyCount[0])
}
