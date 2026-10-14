package websocket

import (
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
	var sum uint64
	for _, b := range payload {
		sum += uint64(b)
	}
	v.EntropyScore.Store(sum)
}
