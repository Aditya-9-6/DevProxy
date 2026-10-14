package buffer

import (
	"testing"
)

func TestTCPGovernor_BDP(t *testing.T) {
	g := NewTCPGovernor()

	// Test default
	if size := g.GetOptimalBufferSize(); size != 65536 {
		t.Errorf("expected default 65536, got %d", size)
	}

	// Record metrics: 1MB in 10ms (10,000,000ns)
	g.RecordMetrics(1024*1024, 10000000)

	size := g.GetOptimalBufferSize()
	if size < 16384 || size > 4194304 {
		t.Errorf("buffer size out of bounds: %d", size)
	}
}

func TestTCPGovernor_Concurrency(t *testing.T) {
	g := NewTCPGovernor()
	stop := make(chan struct{})

	// Concurrent updates
	go func() {
		for i := 0; i < 1000; i++ {
			g.RecordMetrics(1024, 1000)
		}
		close(stop)
	}()

	<-stop

	// Ensure no panic and valid state
	size := g.GetOptimalBufferSize()
	if size == 0 {
		t.Error("expected non-zero buffer size")
	}
}
