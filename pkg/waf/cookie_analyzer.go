package waf

import (
	"github.com/Aditya-9-6/DevProxy/pkg/internal/align"
	"math"
	"sync/atomic"
)

// CookieSecurityEngine tracks session hijacking heuristics.
type CookieSecurityEngine struct {
	ViolationCount align.CacheLinePad
	EntropyScore   align.CacheLinePad
}

func NewCookieSecurityEngine() *CookieSecurityEngine {
	return &CookieSecurityEngine{}
}

// AnalyzeEntropy calculates Shannon entropy of a cookie value.
func (e *CookieSecurityEngine) AnalyzeEntropy(cookie string) float64 {
	if len(cookie) == 0 {
		return 0
	}
	freq := make(map[rune]float64)
	for _, r := range cookie {
		freq[r]++
	}
	entropy := 0.0
	for _, count := range freq {
		p := count / float64(len(cookie))
		entropy -= p * math.Log2(p)
	}
	return entropy
}

// RecordViolation increments the atomic violation counter.
func (e *CookieSecurityEngine) RecordViolation() {
	atomic.AddUint64(&e.ViolationCount[0], 1)
}

// GetViolationCount returns the current count of detected anomalies.
func (e *CookieSecurityEngine) GetViolationCount() uint64 {
	return atomic.LoadUint64(&e.ViolationCount[0])
}
