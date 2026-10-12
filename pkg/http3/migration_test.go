package http3

import (
	"testing"
)

func TestMigrationTracker(t *testing.T) {
	mt := NewMigrationTracker()
	mt.Paths[0].Active.Store(true)
	mt.UpdatePathMetrics(0, 10, 50000)

	if mt.Paths[0].PacketLoss.Load() != 10 {
		t.Errorf("expected loss 10, got %d", mt.Paths[0].PacketLoss.Load())
	}
}

func TestScheduler(t *testing.T) {
	mt := NewMigrationTracker()
	mt.Paths[0].Active.Store(true)
	mt.Paths[0].LatencyNs.Store(100)
	mt.Paths[1].Active.Store(true)
	mt.Paths[1].LatencyNs.Store(50)

	s := NewScheduler(mt)
	path := s.Schedule([]byte("test"))
	if path != 1 {
		t.Errorf("expected path 1, got %d", path)
	}
}
