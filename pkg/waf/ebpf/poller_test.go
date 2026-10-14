package ebpf

import (
	"testing"
)

func TestPollerLifecycle(t *testing.T) {
	p := NewPoller()
	p.Start()
	p.Stop()
}

func TestTelemetryCounters(t *testing.T) {
	tele := &Telemetry{}
	tele.RecordDrop()
	tele.RecordHit()
	drops, hits := tele.GetStats()
	if drops != 1 || hits != 1 {
		t.Errorf("expected 1/1, got %d/%d", drops, hits)
	}
}
