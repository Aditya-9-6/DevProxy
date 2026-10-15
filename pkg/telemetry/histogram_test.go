package telemetry

import (
	"testing"
)

func TestHistogram(t *testing.T) {
	h := &Histogram{}
	h.Record(2000) // 2us
	h.Record(4000) // 4us
	h.Record(8000) // 8us

	p50 := h.GetPercentile(0.5)
	if p50 < 2000 || p50 > 8000 {
		t.Errorf("Expected p50 in range, got %d", p50)
	}
}

func BenchmarkHistogramRecord(b *testing.B) {
	h := &Histogram{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h.Record(5000)
		}
	})
}
