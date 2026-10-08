package proxy

import (
	"sync"
	"testing"
)

func TestLatencyTracker(t *testing.T) {
	lt := &LatencyTracker{}
	// Record 1000 samples: 500 at 10ms, 400 at 50ms, 98 at 100ms, 2 at 500ms
	for i := 0; i < 500; i++ {
		lt.Record(10)
	}
	for i := 0; i < 400; i++ {
		lt.Record(50)
	}
	for i := 0; i < 98; i++ {
		lt.Record(100)
	}
	for i := 0; i < 2; i++ {
		lt.Record(500)
	}

	p50, p90, p99, p999 := lt.GetPercentiles()

	if p50 != 10 {
		t.Errorf("Expected P50 10, got %f", p50)
	}
	if p90 != 50 {
		t.Errorf("Expected P90 50, got %f", p90)
	}
	if p99 != 100 {
		t.Errorf("Expected P99 100, got %f", p99)
	}
	if p999 != 500 {
		t.Errorf("Expected P99.9 500, got %f", p999)
	}
}

func TestLatencyTracker_Empty(t *testing.T) {
	lt := &LatencyTracker{}
	p50, p90, p99, p999 := lt.GetPercentiles()
	if p50 != 0 || p90 != 0 || p99 != 0 || p999 != 0 {
		t.Errorf("Expected all zeros on empty tracker, got %f %f %f %f", p50, p90, p99, p999)
	}
}

func TestLatencyTracker_EdgeCases(t *testing.T) {
	lt := &LatencyTracker{}
	// Negative clamped to 0, overflow clamped to 1000
	lt.Record(-5)
	lt.Record(2000)

	p50, _, _, p999 := lt.GetPercentiles()
	if p50 != 0 {
		t.Errorf("Expected clamped negative to 0, got %f", p50)
	}
	if p999 != 1000 {
		t.Errorf("Expected clamped overflow to 1000, got %f", p999)
	}
}

func TestLatencyTracker_Concurrent(t *testing.T) {
	lt := &LatencyTracker{}
	var wg sync.WaitGroup
	workers := 16
	iterations := 1000

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				lt.Record(int64(i % 200))
			}
		}(w)
	}
	wg.Wait()

	p50, _, _, _ := lt.GetPercentiles()
	if p50 == 0 {
		t.Errorf("Expected non-zero P50 after concurrent recording")
	}
}

func BenchmarkLatencyTracker_Record(b *testing.B) {
	lt := &LatencyTracker{}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i int64
		for pb.Next() {
			lt.Record(i % 500)
			i++
		}
	})
}
