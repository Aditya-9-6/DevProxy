package http2flow

import (
	"sync"
	"sync/atomic"
)

// Ledger tracks HTTP/2 flow control windows for connections and streams.
type Ledger struct {
	connSendWindow  int32
	connRecvWindow  int32
	streamsMu       sync.RWMutex
	streamWindows   map[uint32]*StreamWindow
}

// StreamWindow tracks flow control for an individual HTTP/2 stream.
type StreamWindow struct {
	sendWindow int32
	recvWindow int32
}

// NewLedger creates a new flow control ledger with initial windows.
func NewLedger(initialSendWindow, initialRecvWindow int32) *Ledger {
	return &Ledger{
		connSendWindow: initialSendWindow,
		connRecvWindow: initialRecvWindow,
		streamWindows:  make(map[uint32]*StreamWindow),
	}
}

// GetConnSendWindow returns the current connection send window atomically.
func (l *Ledger) GetConnSendWindow() int32 {
	return atomic.LoadInt32(&l.connSendWindow)
}

// AddConnSendWindow adds delta to the connection send window.
func (l *Ledger) AddConnSendWindow(delta int32) int32 {
	return atomic.AddInt32(&l.connSendWindow, delta)
}
