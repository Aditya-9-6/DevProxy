package security

import (
	"net/http"
	"strings"
	"sync/atomic"
)

// CacheLinePad prevents false sharing on hot atomic counters.
type CacheLinePad [8]uint64

// SmugglingSanitizer tracks potential HTTP smuggling attempts.
type SmugglingSanitizer struct {
	ContradictionCount CacheLinePad
}

func NewSmugglingSanitizer() *SmugglingSanitizer {
	return &SmugglingSanitizer{}
}

// Sanitize checks for dual Content-Length and Transfer-Encoding headers.
func (s *SmugglingSanitizer) Sanitize(headers http.Header) bool {
	cl := headers.Get("Content-Length")
	te := headers.Get("Transfer-Encoding")

	if cl != "" && te != "" {
		atomic.AddUint64(&s.ContradictionCount[0], 1)
		return false
	}

	if te != "" && !strings.Contains(strings.ToLower(te), "chunked") {
		return false
	}

	return true
}
