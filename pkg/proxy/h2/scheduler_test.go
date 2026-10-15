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
	if idx != 0 {
		t.Errorf("Expected stream 1 (weight 20) to be at root, got %d", idx)
	}

	s1.UpdateWeight(s, 50)
	idx = s.SelectStream()
	if idx != 0 {
		t.Errorf("Expected stream 0 (weight 50) to be at root after update, got %d", idx)
	}
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
