package security

import (
	"math"
	"sync"
	"sync/atomic"
)

// CacheLinePaddedScore represents a cache-line padded atomic score tracker
// to avoid false sharing across concurrent CPU cores.
type CacheLinePaddedScore struct {
	_     [7]uint64 // Padding to align with 64-byte cache line boundaries
	score uint64
	_     [7]uint64
}

// Load returns the current entropy anomaly score atomically.
func (s *CacheLinePaddedScore) Load() uint64 {
	return atomic.LoadUint64(&s.score)
}

// Store updates the entropy anomaly score atomically.
func (s *CacheLinePaddedScore) Store(v uint64) {
	atomic.StoreUint64(&s.score, v)
}

// Add adds delta to the entropy anomaly score atomically.
func (s *CacheLinePaddedScore) Add(delta uint64) uint64 {
	return atomic.AddUint64(&s.score, delta)
}

var bufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return &b
	},
}

// CalculateShannonEntropy calculates the Shannon entropy of a given byte sequence
// representing cipher suites and extension identifiers without heap allocations.
func CalculateShannonEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	var freq [256]uint32
	for i := 0; i < len(data); i++ {
		freq[data[i]]++
	}

	var entropy float64
	length := float64(len(data))

	for i := 0; i < 256; i++ {
		if freq[i] == 0 {
			continue
		}
		p := float64(freq[i]) / length
		entropy -= p * math.Log2(p)
	}

	return entropy
}

// AnalyzeTLSClientHello extracts cipher suites and extension bytes and computes entropy score.
func AnalyzeTLSClientHello(clientHello []byte) (float64, uint64) {
	if len(clientHello) < 5 {
		return 0.0, 100 // High anomaly for malformed/truncated hellos
	}

	// Use pooled scratch buffer if needed, keeping hot paths zero-alloc
	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)
	buf := (*bufPtr)[:0]

	// Simple heuristic extraction of handshake payload bytes for entropy
	for _, b := range clientHello {
		buf = append(buf, b)
	}

	entropy := CalculateShannonEntropy(buf)

	// Anomaly score calculation: lower entropy than threshold increases anomaly score
	var anomalyScore uint64
	if entropy < 2.5 {
		anomalyScore = 90
	} else if entropy < 4.0 {
		anomalyScore = 50
	} else {
		anomalyScore = 0
	}

	return entropy, anomalyScore
}
