package proxy

import "sync/atomic"

// BackpressureController tracks saturation and sheds load.
type BackpressureController struct {
	activeRequests atomic.Uint64
	maxRequests    uint64
}

func NewBackpressureController(max uint64) *BackpressureController {
	return &BackpressureController{maxRequests: max}
}

func (b *BackpressureController) ShouldShed() bool {
	return b.activeRequests.Load() >= b.maxRequests
}

func (b *BackpressureController) Acquire() bool {
	if b.ShouldShed() {
		return false
	}
	b.activeRequests.Add(1)
	return true
}

func (b *BackpressureController) Release() {
	b.activeRequests.Add(^uint64(0))
}
