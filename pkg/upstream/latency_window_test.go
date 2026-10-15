package upstream

import (
	"testing"
)

func TestLatencyWindow(t *testing.T) {
	lw := &LatencyWindow{}
	lw.Record(10)
	lw.Record(20)

	avg, count := lw.GetStats()
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
	if avg != 15 {
		t.Errorf("expected avg 15, got %d", avg)
	}
}
