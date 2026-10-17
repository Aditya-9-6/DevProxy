package timeout

import (
	"sync"
	"testing"
	"time"
)

func TestGovernor_InitialState(t *testing.T) {
	g := NewGovernor(100*time.Millisecond, 5*time.Second, 0.2)

	count, avg, ema, variance := g.Stats()
	if count != 0 {
		Errorf(t, "expected initial count 0, got %d", count)
	}
	if avg != 0 {
		Errorf(t, "expected initial avg 0, got %v", avg)
	}
	if ema != 0 {
		Errorf(t, "expected initial ema 0, got %v", ema)
	}
	if variance != 0 {
		Errorf(t, "expected initial variance 0, got %v", variance)
	}

	// Should return minTimeout when no history exists
	if timeout := g.GetDynamicTimeout(); timeout < 100*time.Millisecond {
		Errorf(t, "expected at least minTimeout, got %v", timeout)
	}
}

func TestGovernor_RecordAndAdapt(t *testing.T) {
	g := NewGovernor(50*time.Millisecond, 10*time.Second, 0.3)

	// Record several latencies
	g.RecordLatency(100 * time.Millisecond)
	g.RecordLatency(120 * time.Millisecond)
	g.RecordLatency(110 * time.Millisecond)

	count, _, ema, _ := g.Stats()
	if count != 3 {
		Errorf(t, "expected count 3, got %d", count)
	}
	if ema <= 0 {
		Errorf(t, "expected positive EMA, got %v", ema)
	}

	dynamicTimeout := g.GetDynamicTimeout()
	if dynamicTimeout < 50*time.Millisecond {
		Errorf(t, "dynamic timeout %v below minimum", dynamicTimeout)
	}
}

func TestGovernor_BoundsEnforcement(t *testing.T) {
	g := NewGovernor(200*time.Millisecond, 1*time.Second, 0.5)

	// Record extremely small latency
	g.RecordLatency(1 * time.Millisecond)
	if timeout := g.GetDynamicTimeout(); timeout < 200*time.Millisecond {
		Errorf(t, "timeout %v violated lower bound 200ms", timeout)
	}

	// Record extremely large latency
	g.RecordLatency(50 * time.Second)
	if timeout := g.GetDynamicTimeout(); timeout > 1*time.Second {
		Errorf(t, "timeout %v violated upper bound 1s", timeout)
	}
}

func TestGovernor_ConcurrentSafety(t *testing.T) {
	g := NewGovernor(10*time.Millisecond, 10*time.Second, 0.2)

	var wg sync.WaitGroup
	workers := 10
	iterations := 1000

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				g.RecordLatency(time.Duration(10+j%50) * time.Millisecond)
				_ = g.GetDynamicTimeout()
			}
		}(i)
	}

	wg.Wait()

	count, _, _, _ := g.Stats()
	if count != uint64(workers*iterations) {
		Errorf(t, "expected total count %d, got %d", workers*iterations, count)
	}
}

func Errorf(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	t.Errorf(format, args...)
}
