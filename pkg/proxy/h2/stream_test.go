package h2

import (
	"testing"
	"time"
)

func TestStreamState(t *testing.T) {
	s := NewStreamState(1)
	time.Sleep(10 * time.Millisecond)
	if s.GetIdleDuration() < 10*time.Millisecond {
		t.Errorf("expected idle duration > 10ms, got %v", s.GetIdleDuration())
	}
}

func TestLatencyBuckets(t *testing.T) {
	b := &LatencyBuckets{}
	b.Record(50)
	b.Record(150)
	p := b.GetPercentile(0.5)
	if p < 50 || p > 150 {
		t.Errorf("expected percentile in range, got %d", p)
	}
}
