package tcp

import (
	"testing"
)

func TestWindowEstimator(t *testing.T) {
	estimator := NewWindowEstimator()
	estimator.RecordRTT(50000)
	estimator.RecordRTT(60000)

	cwnd := estimator.EstimateCWND()
	if cwnd == 0 {
		t.Errorf("Expected non-zero CWND, got %d", cwnd)
	}
}

func BenchmarkWindowEstimator(b *testing.B) {
	estimator := NewWindowEstimator()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			estimator.RecordRTT(10000)
			_ = estimator.EstimateCWND()
		}
	})
}
