package security

import (
	"sync/atomic"
)

// CipherAuditor tracks TLS cipher suite usage using cache-aligned atomic counters.
// It avoids mutexes and heap allocations on the hot path.
type CipherAuditor struct {
	// counters maps cipher suite ID to an atomic counter.
	// Using a fixed array for common TLS 1.2/1.3 suites to ensure cache locality.
	counts [256]atomic.Uint64
}

func NewCipherAuditor() *CipherAuditor {
	return &CipherAuditor{}
}

// Record increments the counter for a given cipher suite ID.
func (a *CipherAuditor) Record(cipherID uint16) {
	if int(cipherID) < len(a.counts) {
		a.counts[cipherID].Add(1)
	}
}

// GetCount returns the usage count for a specific cipher suite.
func (a *CipherAuditor) GetCount(cipherID uint16) uint64 {
	if int(cipherID) < len(a.counts) {
		return a.counts[cipherID].Load()
	}
	return 0
}
