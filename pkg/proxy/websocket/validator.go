package websocket

import (
	"math"
	"sync/atomic"
)

type FrameValidator struct {
	ViolationCount atomic.Uint64
	EntropyScore   atomic.Uint64
}

func (v *FrameValidator) ValidateMask(mask [4]byte) bool {
	if mask[0] == 0 && mask[1] == 0 && mask[2] == 0 && mask[3] == 0 {
		v.ViolationCount.Add(1)
		return false
	}
	return true
}

func (v *FrameValidator) TrackEntropy(payload []byte) {
	if len(payload) == 0 {
		v.EntropyScore.Store(0)
		return
	}

	var counts [256]int
	for _, b := range payload {
		counts[b]++
	}

	var entropy float64
	length := float64(len(payload))
	for _, count := range counts {
		if count > 0 {
			p := float64(count) / length
			entropy -= p * math.Log2(p)
		}
	}

	score := uint64(entropy * 1000)
	v.EntropyScore.Store(score)
}
