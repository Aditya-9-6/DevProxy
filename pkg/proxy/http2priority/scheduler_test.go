package http2priority

import (
	"testing"
)

func TestSchedulerTree(t *testing.T) {
	s := NewScheduler()
	if s.Root == nil {
		t.Fatal("expected non-nil root node")
	}

	s.Insert(1, 0, 32, false)
	if len(s.Root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(s.Root.Children))
	}
}
