package h3flow

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to 64 bytes.
type CacheLinePad [8]uint64

// FlowControl tracks HTTP/3 datagram flow state without locks.
type FlowControl struct {
	WindowSize    atomic.Uint64
	InFlightBytes atomic.Uint64
	PacketLoss    atomic.Uint64
}

func NewFlowControl(initialWindow uint64) *FlowControl {
	fc := &FlowControl{}
	fc.WindowSize.Store(initialWindow)
	return fc
}

func (f *FlowControl) CanSend(size uint64) bool {
	return f.InFlightBytes.Load()+size <= f.WindowSize.Load()
}
