package http2header

import (
	"sync"
	"sync/atomic"
)

type PressureAccountant struct {
	totalBytes int64
	maxAllowed int64
	mu         sync.Mutex
}

func NewPressureAccountant(maxAllowed int64) *PressureAccountant {
	return &PressureAccountant{
		maxAllowed: maxAllowed,
	}
}

func (p *PressureAccountant) AddBytes(n int64) bool {
	newTotal := atomic.AddInt64(&p.totalBytes, n)
	return newTotal <= p.maxAllowed
}

func (p *PressureAccountant) Reset() {
	atomic.StoreInt64(&p.totalBytes, 0)
}

func (p *PressureAccountant) Total() int64 {
	return atomic.LoadInt64(&p.totalBytes)
}
