package upstream

import (
	"sync"
	"testing"
)

func TestLatencyWindow(t *testing.T) {
	lw := &LatencyWindow{}
	lw.Record(10)
	lw.Record(20)

	avg, count := lw.GetStats()
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
	if avg != 15 {
		t.Errorf("expected avg 15, got %d", avg)
	}
}

func TestLatencyWindow_Concurrent(t *testing.T) {
	lw := &LatencyWindow{}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				lw.Record(uint64(j))
				_, _ = lw.GetStats()
			}
		}()
	}
	wg.Wait()
}
