package upstream

import (
	"context"
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
	return NewPassiveMonitorWithContext(context.Background(), thresholdMs, 50*time.Millisecond)
}

func NewPassiveMonitorWithContext(ctx context.Context, thresholdMs uint64, interval time.Duration) *PassiveMonitor {
	return &PassiveMonitor{
		window:      NewLatencyWindow(ctx, interval),
		thresholdMs: thresholdMs,
	}
}

func (pm *PassiveMonitor) Close() {
	if pm.window != nil {
		pm.window.Close()
	}
}

func (pm *PassiveMonitor) Observe(start time.Time) {
	ms := time.Since(start).Milliseconds()
	var duration uint64
	if ms > 0 {
		// #nosec G115 -- ms is verified to be positive
		duration = uint64(ms) // #nosec G115
	}
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
