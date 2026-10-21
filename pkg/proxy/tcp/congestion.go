package tcp

import (
	"sync/atomic"
)

const (
	WindowSize = 1024
)

type WindowEstimator struct {
	_          [64]byte
	timestamps [WindowSize]atomic.Uint64
	_          [64]byte
	runningSum atomic.Uint64
	_          [64]byte
	count      atomic.Uint64
	_          [64]byte
	cursor     atomic.Uint64
	_          [64]byte
}

func NewWindowEstimator() *WindowEstimator {
	return &WindowEstimator{}
}

func (e *WindowEstimator) RecordRTT(rttNs uint64) {
	idx := e.cursor.Add(1) % WindowSize
	old := e.timestamps[idx].Swap(rttNs)

	if old == 0 {
		e.count.Add(1)
	} else {
		e.runningSum.Add(^uint64(old - 1))
	}
	e.runningSum.Add(rttNs)
}

func (e *WindowEstimator) EstimateCWND() uint64 {
	count := e.count.Load()
	if count == 0 {
		return 10
	}
	sum := e.runningSum.Load()
	avg := sum / count
	return 1000000 / (avg/1000 + 1)
}
