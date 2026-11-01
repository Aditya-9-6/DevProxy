package timeout

import (
	"sync"
	"testing"
	"time"
)

func TestGovernor_NormalOperation(t *testing.T) {
	g := NewGovernor(50*time.Millisecond, 5*time.Second, 3.0, 0.2)

	// Initial timeout should be within bounds
	initial := g.GetDynamicTimeout()
	if initial < 50*time.Millisecond || initial > 5*time.Second {
		t.Errorf("initial timeout %v out of bounds", initial)
	}

	// Record several consistent latencies
	for i := 0; i < 10; i++ {
		g.RecordLatency(100 * time.Millisecond)
	}

	dynamic := g.GetDynamicTimeout()
	if dynamic < 50*time.Millisecond {
		t.Errorf("dynamic timeout %v fell below minimum", dynamic)
	}
}

func TestGovernor_BoundaryConstraints(t *testing.T) {
	g := NewGovernor(100*time.Millisecond, 200*time.Millisecond, 3.0, 0.5)

	// Record extremely low latencies
	for i := 0; i < 5; i++ {
		g.RecordLatency(1 * time.Millisecond)
	}
	if d := g.GetDynamicTimeout(); d < 100*time.Millisecond {
		t.Errorf("timeout %v violated min bound 100ms", d)
	}

	// Record extremely high latencies
	for i := 0; i < 5; i++ {
		g.RecordLatency(10 * time.Second)
	}
	if d := g.GetDynamicTimeout(); d > 200*time.Millisecond {
		t.Errorf("timeout %v violated max bound 200ms", d)
	}
}

func TestGovernor_ConcurrencySafety(t *testing.T) {
	g := NewGovernor(10*time.Millisecond, 10*time.Second, 3.0, 0.2)

	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				g.RecordLatency(time.Duration(j+1) * time.Millisecond)
				_ = g.GetDynamicTimeout()
			}
		}(i)
	}

	wg.Wait()

	// Ensure reset works cleanly under no contention
	g.Reset()
	if g.GetDynamicTimeout() < 10*time.Millisecond {
		t.Errorf("reset failed to restore min timeout boundary")
	}
}
