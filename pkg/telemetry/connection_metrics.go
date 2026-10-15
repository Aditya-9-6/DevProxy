package telemetry

import (
	"sync/atomic"
)

// CacheLinePad ensures 64-byte alignment to prevent false sharing on CPU caches.
type CacheLinePad [8]uint64

// ConnectionMetrics tracks HTTP/1.1 and HTTP/2 connection lifecycle events.
// All counters are atomic and cache-line aligned.
type ConnectionMetrics struct {
	TotalConnections  CacheLinePad
	ActiveConnections CacheLinePad
	ReusedConnections CacheLinePad
	TotalIdleDuration CacheLinePad
	TotalRequests     CacheLinePad
}

// NewConnectionMetrics initializes a new telemetry collector.
func NewConnectionMetrics() *ConnectionMetrics {
	return &ConnectionMetrics{}
}

// RecordNewConnection increments the total connection counter.
func (m *ConnectionMetrics) RecordNewConnection() {
	atomic.AddUint64(&m.TotalConnections[0], 1)
	atomic.AddUint64(&m.ActiveConnections[0], 1)
}

// RecordReuse increments the reuse counter.
func (m *ConnectionMetrics) RecordReuse() {
	atomic.AddUint64(&m.ReusedConnections[0], 1)
}

// RecordClose decrements the active connection counter.
func (m *ConnectionMetrics) RecordClose() {
	atomic.AddUint64(&m.ActiveConnections[0], ^uint64(0))
}

// RecordRequest increments the request count per connection.
func (m *ConnectionMetrics) RecordRequest() {
	atomic.AddUint64(&m.TotalRequests[0], 1)
}

// RecordIdleDuration adds to the cumulative idle time.
func (m *ConnectionMetrics) RecordIdleDuration(ns uint64) {
	atomic.AddUint64(&m.TotalIdleDuration[0], ns)
}

// GetStats returns the current snapshot of metrics without heap allocations.
func (m *ConnectionMetrics) GetStats() (total, active, reused, requests, idle uint64) {
	return atomic.LoadUint64(&m.TotalConnections[0]),
		atomic.LoadUint64(&m.ActiveConnections[0]),
		atomic.LoadUint64(&m.ReusedConnections[0]),
		atomic.LoadUint64(&m.TotalRequests[0]),
		atomic.LoadUint64(&m.TotalIdleDuration[0])
}
