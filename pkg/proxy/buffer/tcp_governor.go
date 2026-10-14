package buffer

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to 64 bytes.
type CacheLinePad [8]uint64

// TCPGovernor manages dynamic SO_RCVBUF sizing using lock-free atomic metrics.
type TCPGovernor struct {
	// BytesReceived tracks total bytes processed for BDP calculation.
	BytesReceived CacheLinePad
	// LatencyNs tracks rolling average latency in nanoseconds.
	LatencyNs CacheLinePad
}

// NewTCPGovernor initializes a new governor.
func NewTCPGovernor() *TCPGovernor {
	return &TCPGovernor{}
}

// RecordMetrics updates the governor with throughput and latency data.
func (g *TCPGovernor) RecordMetrics(bytes uint64, latencyNs uint64) {
	atomic.AddUint64(&g.BytesReceived[0], bytes)

	// Exponential Moving Average for latency: (old * 0.9) + (new * 0.1)
	for {
		old := atomic.LoadUint64(&g.LatencyNs[0])
		newScore := (old*9 + latencyNs) / 10
		if atomic.CompareAndSwapUint64(&g.LatencyNs[0], old, newScore) {
			break
		}
	}
}

// GetOptimalBufferSize calculates the BDP (Bandwidth * Delay) to suggest a buffer size.
// Returns a value in bytes, clamped between 16KB and 4MB.
func (g *TCPGovernor) GetOptimalBufferSize() int {
	bytes := atomic.LoadUint64(&g.BytesReceived[0])
	latency := atomic.LoadUint64(&g.LatencyNs[0])

	if latency == 0 || bytes == 0 {
		return 65536 // Default 64KB
	}

	// BDP = Bandwidth (bytes/ns) * Latency (ns)
	// Simplified: (bytes / time_window) * latency
	// We use a conservative estimate for buffer sizing.
	bdp := (bytes / 1000) * (latency / 1000)

	if bdp < 16384 {
		return 16384
	}
	if bdp > 4194304 {
		return 4194304
	}
	return int(bdp)
}
