package proxy

import (
	"sync/atomic"
	"time"
)

// Mitigator tracks socket states using lock-free atomic counters.
type Mitigator struct {
	ActiveConnections   CacheLinePad
	SlowLorisViolations CacheLinePad
	ReadDeadline        atomic.Int64
}

// NewMitigator initializes a new socket mitigation engine.
func NewMitigator() *Mitigator {
	return &Mitigator{}
}

// TrackConnection increments the active connection count.
func (m *Mitigator) TrackConnection() {
	atomic.AddUint64(&m.ActiveConnections[0], 1)
}

// ReleaseConnection decrements the active connection count.
func (m *Mitigator) ReleaseConnection() {
	atomic.AddUint64(&m.ActiveConnections[0], ^uint64(0))
}

// SetReadDeadline sets the absolute deadline for the next read operation.
func (m *Mitigator) SetReadDeadline(d time.Time) {
	m.ReadDeadline.Store(d.UnixNano())
}

// IsExpired checks if the current connection has exceeded its read deadline.
func (m *Mitigator) IsExpired() bool {
	deadline := m.ReadDeadline.Load()
	if deadline == 0 {
		return false
	}
	return time.Now().UnixNano() > deadline
}

// RecordViolation increments the slow-loris violation counter.
func (m *Mitigator) RecordViolation() {
	atomic.AddUint64(&m.SlowLorisViolations[0], 1)
}
