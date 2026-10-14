package tcp

import (
	"sync/atomic"
)

const (
	WindowSize = 1024
	CacheLine  = 64
)

type CacheLinePad [8]uint64

type WindowEstimator struct {
	_          [CacheLine]byte
	timestamps [WindowSize]atomic.Uint64
	_          [CacheLine]byte
	cursor     atomic.Uint64
	_          [CacheLine]byte
}

func NewWindowEstimator() *WindowEstimator {
	return &WindowEstimator{}
}

func (e *WindowEstimator) RecordRTT(rttNs uint64) {
	idx := e.cursor.Add(1) % WindowSize
	e.timestamps[idx].Store(rttNs)
}

func (e *WindowEstimator) EstimateCWND() uint64 {
	var total uint64
	var count uint64
	for i := 0; i < WindowSize; i++ {
		val := e.timestamps[i].Load()
		if val > 0 {
			total += val
			count++
		}
	}
	if count == 0 {
		return 10
	}
	avg := total / count
	return 1000000 / (avg/1000 + 1)
}
