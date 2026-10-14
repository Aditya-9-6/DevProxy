package security

import (
	"sync/atomic"
	"time"
)

const (
	WindowSize = 10
	MaxFrames  = 100
)

type bucket struct {
	count uint64
	_     [56]byte // Pad to 64 bytes to prevent false sharing
}

type ContinuationDetector struct {
	buckets    [WindowSize]bucket
	totalCount atomic.Uint64
	lastIdx    atomic.Int64
}

func NewContinuationDetector() *ContinuationDetector {
	return &ContinuationDetector{}
}

func (d *ContinuationDetector) RecordFrame() bool {
	now := time.Now().Unix()
	idx := now % WindowSize

	last := d.lastIdx.Load()
	if now > last {
		d.rotate(last, now)
		d.lastIdx.Store(now)
	}

	atomic.AddUint64(&d.buckets[idx].count, 1)
	return d.totalCount.Add(1) > MaxFrames
}

func (d *ContinuationDetector) rotate(last, now int64) {
	diff := now - last
	if diff >= WindowSize {
		for i := 0; i < WindowSize; i++ {
			val := atomic.SwapUint64(&d.buckets[i].count, 0)
			if val > 0 {
				d.totalCount.Add(^uint64(val - 1))
			}
		}
		return
	}

	for i := int64(1); i <= diff; i++ {
		idx := (last + i) % WindowSize
		val := atomic.SwapUint64(&d.buckets[idx].count, 0)
		if val > 0 {
			d.totalCount.Add(^uint64(val - 1))
		}
	}
}
