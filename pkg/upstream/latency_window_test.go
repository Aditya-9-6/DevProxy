package upstream

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestLatencyWindow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lw := NewLatencyWindow(ctx, 10*time.Millisecond)
	defer lw.Close()

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lw := NewLatencyWindow(ctx, 10*time.Millisecond)
	defer lw.Close()

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

func TestLatencyWindow_LifecycleShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	lw := NewLatencyWindow(ctx, 5*time.Millisecond)

	// Record some values
	lw.Record(50)

	// Close via context and explicit Close method
	cancel()
	lw.Close()

	// Verify no goroutine leak (test passing without hanging is proof)
}

func TestLatencyWindow_EmptyStats(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lw := NewLatencyWindow(ctx, 0)
	defer lw.Close()

	avg, count := lw.RefreshStats()
	if avg != 0 || count != 0 {
		t.Errorf("expected 0, 0 for empty stats, got %d, %d", avg, count)
	}

	avg, count = lw.GetStats()
	if avg != 0 || count != 0 {
		t.Errorf("expected cached stats 0, 0, got %d, %d", avg, count)
	}
}

func TestPassiveMonitor_HealthStateTransition(t *testing.T) {
	pm := NewPassiveMonitor(50) // threshold 50ms
	defer pm.Close()

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

func TestPassiveMonitor_EdgeCases(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pm := NewPassiveMonitorWithContext(ctx, 50, 10*time.Millisecond)
	defer pm.Close()

	// Observe future start time (negative delta ms)
	pm.Observe(time.Now().Add(10 * time.Second))
	if !pm.IsHealthy() {
		t.Errorf("expected node to remain healthy on negative delta")
	}

	// Multiple Close calls should not panic
	pm.Close()
}
