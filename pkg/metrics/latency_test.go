package metrics

import (
	"testing"
	"time"
)

func TestLatencyTracker(t *testing.T) {
	tracker := NewLatencyTracker()

	// Record some latencies
	tracker.Record(10 * time.Millisecond)
	tracker.Record(100 * time.Millisecond)
	tracker.Record(500 * time.Millisecond)

	var p50, p90, p99 float64
	tracker.GetPercentiles(&p50, &p90, &p99)

	if p50 == 0 || p90 == 0 || p99 == 0 {
		t.Errorf("Percentiles not calculated correctly: p50=%f, p90=%f, p99=%f", p50, p90, p99)
	}
}

func TestLatencyTrackerBoundary(t *testing.T) {
	tracker := NewLatencyTracker()
	tracker.Record(2000 * time.Millisecond) // Should cap at 999
	tracker.Record(-1 * time.Millisecond)   // Should floor at 0

	var p50 float64
	tracker.GetPercentiles(&p50, nil, nil)
	if p50 < 0 || p50 >= MaxLatencyMs {
		t.Errorf("Out of bounds latency handling failed: p50=%f", p50)
	}
}
