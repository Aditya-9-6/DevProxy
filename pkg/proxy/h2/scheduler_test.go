package h2

import (
	"sync"
	"testing"
)

func TestScheduler_FairShare(t *testing.T) {
	s := NewScheduler()
	s1 := s.AddStream(10)
	_ = s.AddStream(20)

	idx := s.SelectStream()
	s.mu.RLock()
	rootStream := s.streams[0]
	s.mu.RUnlock()
	if rootStream.Weight != 20 {
		t.Errorf("Expected stream with weight 20 to be at root, got %d", rootStream.Weight)
	}

	s1.UpdateWeight(s, 50)
	idx = s.SelectStream()
	s.mu.RLock()
	rootStream = s.streams[0]
	s.mu.RUnlock()
	if rootStream.Weight != 50 {
		t.Errorf("Expected stream with weight 50 to be at root after update, got %d", rootStream.Weight)
	}
	_ = idx
}

func TestScheduler_Empty(t *testing.T) {
	s := NewScheduler()
	if idx := s.SelectStream(); idx != -1 {
		t.Errorf("Expected -1 for empty scheduler, got %d", idx)
	}
}

func TestScheduler_Concurrent(t *testing.T) {
	s := NewScheduler()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val uint64) {
			defer wg.Done()
			sw := s.AddStream(val)
			sw.UpdateWeight(s, val+10)
			s.SelectStream()
		}(uint64(i))
	}
	wg.Wait()
}

func TestScheduler_HeapIntegrity(t *testing.T) {
	s := NewScheduler()
	streams := make([]*StreamWeight, 10)
	for i := 0; i < 10; i++ {
		streams[i] = s.AddStream(uint64(i))
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			streams[idx].UpdateWeight(s, 1000-uint64(idx))
		}(i)
	}
	wg.Wait()

	s.mu.RLock()
	rootWeight := s.streams[0].Weight
	s.mu.RUnlock()
	if rootWeight != 1000 {
		t.Errorf("Heap property violated: root weight is %d, expected 1000", rootWeight)
	}
}

func BenchmarkSelectStream(b *testing.B) {
	s := NewScheduler()
	for i := 0; i < 1000; i++ {
		s.AddStream(uint64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SelectStream()
	}
}
