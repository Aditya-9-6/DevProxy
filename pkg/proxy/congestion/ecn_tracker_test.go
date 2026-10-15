package congestion

import (
	"testing"
)

func TestECNTracker(t *testing.T) {
	tracker := NewECNTracker()
	tracker.RecordCongestion()
	tracker.RecordCongestion()

	if tracker.GetCongestionScore() != 2 {
		t.Errorf("expected score 2, got %d", tracker.GetCongestionScore())
	}

	tracker.DecayCongestion(2)
	if tracker.GetCongestionScore() != 1 {
		t.Errorf("expected decayed score 1, got %d", tracker.GetCongestionScore())
	}
}
