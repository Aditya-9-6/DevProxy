package telemetry

import (
	"testing"
)

func TestConnectionMetrics(t *testing.T) {
	m := NewConnectionMetrics()

	m.RecordNewConnection()
	m.RecordNewConnection()
	m.RecordReuse()
	m.RecordRequest()
	m.RecordIdleDuration(100)
	m.RecordClose()

	total, active, reused, requests, idle := m.GetStats()

	if total != 2 {
		t.Errorf("expected 2 total connections, got %d", total)
	}
	if active != 1 {
		t.Errorf("expected 1 active connection, got %d", active)
	}
	if reused != 1 {
		t.Errorf("expected 1 reused connection, got %d", reused)
	}
	if requests != 1 {
		t.Errorf("expected 1 request, got %d", requests)
	}
	if idle != 100 {
		t.Errorf("expected 100ns idle, got %d", idle)
	}
}
