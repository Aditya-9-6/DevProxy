package proxy

import (
	"sync/atomic"
	"time"
)

const (
	WheelSize = 1024
	TickMs    = 100
)

type wheelBucket struct {
	_     [56]byte
	count atomic.Uint64
	_     [56]byte
}

type TimingWheel struct {
	buckets [WheelSize]wheelBucket
	cursor  atomic.Uint64
}

func NewTimingWheel() *TimingWheel {
	return &TimingWheel{}
}

func (tw *TimingWheel) Register(deadline time.Duration) {
	idx := (tw.cursor.Load() + uint64(deadline.Milliseconds()/TickMs)) % WheelSize
	tw.buckets[idx].count.Add(1)
}

func (tw *TimingWheel) Tick() {
	tw.cursor.Add(1)
}

func (tw *TimingWheel) GetExpiredCount() uint64 {
	idx := tw.cursor.Load() % WheelSize
	return tw.buckets[idx].count.Swap(0)
}
