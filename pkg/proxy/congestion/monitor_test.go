package congestion

import (
	"testing"
)

func TestMonitorAndWindow(t *testing.T) {
	monitor := NewMonitor(10)

	if monitor.IsCongested() {
		data, _ := monitor.Stats()
		if data != 0 {
			t.Errorf("expected 0 packets, got %d", data)
		}
	}

	monitor.RecordPacket(false)
	if monitor.IsCongested() {
		t.Errorf("expected not congested for normal packet")
	}

	monitor.RecordPacket(true)
	if !monitor.IsCongested() {
		t.Errorf("expected congested after ECN-CE packet")
	}

	total, ce := monitor.Stats()
	if total != 2 || ce != 1 {
		t.Errorf("unexpected stats: total=%d, ce=%d", total, ce)
	}

	win := monitor.Window()
	if win.CurrentWindow() != 0 {
		t.Errorf("expected window 0 after congestion, got %d", win.CurrentWindow())
	}
}
