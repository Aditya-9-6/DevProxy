package balancer

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to 64 bytes.
type CacheLinePad [8]uint64

// TargetStats tracks performance and load metrics for an upstream target.
type TargetStats struct {
	ActiveConnections CacheLinePad
	LatencyScore      CacheLinePad
}

// RecordConnection increments the active connection count.
func (s *TargetStats) RecordConnection() {
	atomic.AddUint64(&s.ActiveConnections[0], 1)
}

// ReleaseConnection decrements the active connection count.
func (s *TargetStats) ReleaseConnection() {
	atomic.AddUint64(&s.ActiveConnections[0], ^uint64(0))
}

// UpdateLatency updates the rolling latency score using an exponential moving average.
func (s *TargetStats) UpdateLatency(newLatency uint64) {
	for {
		old := atomic.LoadUint64(&s.LatencyScore[0])
		// EMA: score = (old * 0.9) + (new * 0.1)
		newScore := (old*9 + newLatency) / 10
		if atomic.CompareAndSwapUint64(&s.LatencyScore[0], old, newScore) {
			break
		}
	}
}

// GetScore returns the combined load pressure score.
func (s *TargetStats) GetScore() uint64 {
	return atomic.LoadUint64(&s.ActiveConnections[0])*1000 + atomic.LoadUint64(&s.LatencyScore[0])
}
