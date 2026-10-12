package http2

import (
	"sync"
	"testing"
	"time"
)

func TestLatencyTracker(t *testing.T) {
	tracker := &LatencyTracker{}
	tracker.Record(10 * time.Microsecond)
	tracker.Record(50 * time.Microsecond)

	p50 := tracker.GetPercentile(0.5)
	if p50 < 10*time.Microsecond || p50 > 50*time.Microsecond {
		t.Errorf("Expected p50 in range, got %v", p50)
	}
}

func TestSchedulerPool(t *testing.T) {
	s := NewScheduler()
	p := s.GetPriority()
	if p == nil {
		t.Fatal("Expected priority object")
	}
	s.PutPriority(p)
}

func TestDependencyTreeConcurrency(t *testing.T) {
	t.Parallel()
	tree := NewDependencyTree()
	var wg sync.WaitGroup

	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(id uint32) {
			defer wg.Done()
			tree.AddChild(0, id, 16)
		}(uint32(i))
	}
	wg.Wait()
}

func BenchmarkDependencyTreeAdd(b *testing.B) {
	tree := NewDependencyTree()
	b.RunParallel(func(pb *testing.PB) {
		i := uint32(1)
		for pb.Next() {
			tree.AddChild(0, i, 16)
			i++
		}
	})
}
