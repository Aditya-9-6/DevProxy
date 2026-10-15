package h2

import (
	"testing"
)

func TestScheduler_FairShare(t *testing.T) {
	s := NewScheduler()
	s1 := s.AddStream(10)
	_ = s.AddStream(20)

	idx := s.SelectStream()
	if idx != 1 {
		t.Errorf("Expected stream 1 (weight 20) to be selected, got %d", idx)
	}

	s1.UpdateWeight(50)
	idx = s.SelectStream()
	if idx != 0 {
		t.Errorf("Expected stream 0 (weight 50) to be selected after update, got %d", idx)
	}
}
