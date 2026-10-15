package upstream

import (
	"sync"
	"testing"
)

func TestSelectorBasicLeastConnections(t *testing.T) {
	addrs := []string{"backend-1:8080", "backend-2:8080", "backend-3:8080"}
	sel := NewSelector(addrs)

	t1 := sel.Select()
	if t1 == nil {
		t.Fatal("expected non-nil target")
	}

	// Simulate active streams on backend-1 and backend-2
	t1.Stats.IncStreams()
	t2 := sel.Select()
	if t2.Addr == t1.Addr {
		t.Errorf("expected different backend since %s has active stream", t1.Addr)
	}

	t2.Stats.IncStreams()
	t2.Stats.IncStreams()

	// Now backend-3 should have 0 active streams and be selected
	t3 := sel.Select()
	if t3.Addr != "backend-3:8080" {
		t.Errorf("expected backend-3:8080, got %s", t3.Addr)
	}

	t3.Stats.IncStreams()
	t3.Stats.DecStreams()
	if val := t3.Stats.GetActiveStreams(); val != 0 {
		t.Errorf("expected active streams to be 0, got %d", val)
	}
}

func TestAtomicTargetStatsConcurrency(t *testing.T) {
	stats := NewAtomicTargetStats()
	var wg sync.WaitGroup
	workers := 50
	its := 200

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < its; j++ {
				stats.IncStreams()
				stats.GetActiveStreams()
				stats.DecStreams()
			}
		}()
	}

	wg.Wait()

	if active := stats.GetActiveStreams(); active != 0 {
		t.Errorf("expected 0 active streams after concurrent increments and decrements, got %d", active)
	}

	expectedTotal := uint64(workers * its)
	if total := stats.GetTotalRequests(); total != expectedTotal {
		t.Errorf("expected total requests %d, got %d", expectedTotal, total)
	}
}

func TestSelectorDynamicAddAndEmpty(t *testing.T) {
	emptySel := NewSelector(nil)
	if emptySel.Select() != nil {
		t.Error("expected nil for empty selector")
	}

	emptySel.AddTarget("dynamic-node:9000")
	chosen := emptySel.Select()
	if chosen == nil || chosen.Addr != "dynamic-node:9000" {
		t.Errorf("expected dynamic-node:9000, got %v", chosen)
	}
}
