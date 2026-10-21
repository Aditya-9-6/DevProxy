package tcp

import (
	"sync"
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

func TestWindowEstimatorConcurrent(t *testing.T) {
	estimator := NewWindowEstimator()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				estimator.RecordRTT(uint64(j))
				_ = estimator.EstimateCWND()
			}
		}()
	}
	wg.Wait()
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
