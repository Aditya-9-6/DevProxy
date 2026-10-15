package tls

import (
	"math"
)

// AnalyzePaddingEntropy calculates the Shannon entropy of a byte slice.
// Used to detect non-random padding patterns in encrypted TLS records.
func AnalyzePaddingEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	counts := make([]int, 256)
	for _, b := range data {
		counts[b]++
	}

	var entropy float64
	dataLen := float64(len(data))
	for _, count := range counts {
		if count > 0 {
			p := float64(count) / dataLen
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}
