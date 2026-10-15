package security

import (
	"math"
	"sync/atomic"
)

// CacheLinePad is used to prevent false sharing in high-concurrency structures.
type CacheLinePad [8]uint64

type EntropyScore struct {
	Score CacheLinePad
}

func (e *EntropyScore) UpdateScore(newScore uint64) {
	atomic.StoreUint64(&e.Score[0], newScore)
}

func (e *EntropyScore) GetScore() uint64 {
	return atomic.LoadUint64(&e.Score[0])
}

func CalculateShannonEntropy(data []uint16) float64 {
	if len(data) == 0 {
		return 0
	}
	freq := make(map[uint16]int)
	for _, v := range data {
		freq[v]++
	}
	var entropy float64
	ln2 := math.Log(2)
	for _, count := range freq {
		p := float64(count) / float64(len(data))
		entropy -= p * (math.Log(p) / ln2)
	}
	return entropy
}
