package http2

import (
	"testing"
	"time"
)

func TestLatencyTracker(t *testing.T) {
	tracker := &LatencyTracker{}
	tracker.Record(10 * time.Microsecond)
	tracker.Record(50 * time.Microsecond)

	p50 := tracker.GetPercentile(0.5)
	if p50 < 10*time.Microsecond || p50 > 50*time.Microsecond {
		t.Errorf("Expected p50 in range, got %v", p50)
	}
}

func TestSchedulerPool(t *testing.T) {
	s := NewScheduler()
	p := s.GetPriority()
	if p == nil {
		t.Fatal("Expected priority object")
	}
	s.PutPriority(p)
}
