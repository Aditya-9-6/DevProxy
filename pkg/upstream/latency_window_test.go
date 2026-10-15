package upstream

import (
	"sync"
	"testing"
	"time"
)

func TestLatencyWindow(t *testing.T) {
	lw := NewLatencyWindow()
	lw.Record(10)
	lw.Record(20)

	avg, count := lw.RefreshStats()
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
	if avg != 15 {
		t.Errorf("expected avg 15, got %d", avg)
	}

	cachedAvg, cachedCount := lw.GetStats()
	if cachedCount != 2 || cachedAvg != 15 {
		t.Errorf("cached stats mismatch: avg=%d, count=%d", cachedAvg, cachedCount)
	}
}

func TestLatencyWindow_Concurrent(t *testing.T) {
	lw := NewLatencyWindow()
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

func TestPassiveMonitor_HealthStateTransition(t *testing.T) {
	pm := NewPassiveMonitor(50) // threshold 50ms

	// Initially healthy
	if !pm.IsHealthy() {
		t.Errorf("expected node to be healthy initially")
	}

	// Observe low latencies
	pm.Observe(time.Now().Add(-10 * time.Millisecond))
	if !pm.IsHealthy() {
		t.Errorf("expected node to remain healthy with 10ms latency")
	}

	// Observe high latencies exceeding threshold
	pm.Observe(time.Now().Add(-100 * time.Millisecond))
	if pm.IsHealthy() {
		t.Errorf("expected node to transition to Degraded when latency exceeds threshold")
	}
}
