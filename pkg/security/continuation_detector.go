package security

import (
	"sync/atomic"
	"time"
)

const (
	WindowSize = 10
	MaxFrames  = 100
)

type CacheLinePad [8]uint64

type ContinuationDetector struct {
	buckets [WindowSize]CacheLinePad
}

func NewContinuationDetector() *ContinuationDetector {
	return &ContinuationDetector{}
}

func (d *ContinuationDetector) RecordFrame() bool {
	idx := (time.Now().Unix() % WindowSize)
	atomic.AddUint64(&d.buckets[idx][0], 1)

	var total uint64
	for i := 0; i < WindowSize; i++ {
		total += atomic.LoadUint64(&d.buckets[i][0])
	}
	return total > MaxFrames
}

func (d *ContinuationDetector) ResetBucket(ts int64) {
	idx := (ts % WindowSize)
	atomic.StoreUint64(&d.buckets[idx][0], 0)
}
