package upstream

import (
	"sync/atomic"
	"time"
)

// HealthStatus represents the health of an upstream node.
type HealthStatus int32

const (
	Healthy HealthStatus = iota
	Degraded
)

// PassiveMonitor tracks upstream health based on handshake latency.
type PassiveMonitor struct {
	window      *LatencyWindow
	status      atomic.Int32
	thresholdMs uint64
}

func NewPassiveMonitor(thresholdMs uint64) *PassiveMonitor {
	return &PassiveMonitor{
		window:      NewLatencyWindow(),
		thresholdMs: thresholdMs,
	}
}

func (pm *PassiveMonitor) Observe(start time.Time) {
	duration := uint64(time.Since(start).Milliseconds())
	pm.window.Record(duration)

	// Force refresh stats for immediate evaluation in tests/observers if needed
	avg, count := pm.window.RefreshStats()
	if count > 0 && avg > pm.thresholdMs {
		pm.status.Store(int32(Degraded))
	} else {
		pm.status.Store(int32(Healthy))
	}
}

func (pm *PassiveMonitor) IsHealthy() bool {
	return pm.status.Load() == int32(Healthy)
}
