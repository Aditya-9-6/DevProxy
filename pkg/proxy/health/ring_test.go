package health

import (
	"sync"
	"testing"
)

func TestHealthRing_SuccessRate(t *testing.T) {
	ring := NewHealthRing()

	ring.Record(true, 100)
	ring.Record(true, 200)
	ring.Record(false, 0)
	ring.Record(true, 150)
	ring.Record(false, 0)

	expected := 3.0 / 16.0
	if got := ring.SuccessRate(); got != expected {
		t.Errorf("SuccessRate() = %v, want %v", got, expected)
	}
}

func TestHealthRing_Concurrent(t *testing.T) {
	ring := NewHealthRing()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := uint64(0); j < 100; j++ {
				ring.Record(j%2 == 0, j*10)
			}
		}()
	}

	wg.Wait()

	rate := ring.SuccessRate()
	if rate < 0 || rate > 1 {
		t.Errorf("SuccessRate() out of range: %v", rate)
	}
}

func TestHealthRing_ConcurrentRecordsPreserveAllBits(t *testing.T) {
	ring := NewHealthRing()
	start := make(chan struct{})

	var wg sync.WaitGroup
	for i := 0; i < healthRingSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ring.Record(true, 1)
		}()
	}

	close(start)
	wg.Wait()

	if rate := ring.SuccessRate(); rate != 1 {
		t.Errorf("SuccessRate() = %v, want 1", rate)
	}
}

func TestHealthRing_AvgLatencyDoesNotOverflow(t *testing.T) {
	ring := NewHealthRing()
	maxLatency := ^uint64(0)

	for i := 0; i < healthRingSize; i++ {
		ring.Record(true, maxLatency)
	}

	expected := float64(maxLatency)
	if got := ring.AvgLatency(); got != expected {
		t.Errorf("AvgLatency() = %v, want %v", got, expected)
	}
}

func TestHealthRing_LatenciesReturnsSnapshot(t *testing.T) {
	ring := NewHealthRing()
	ring.Record(true, 42)

	snapshot := ring.Latencies()
	ring.Record(true, 99)

	found := false
	for _, latency := range snapshot {
		if latency == 42 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected latency snapshot to retain previously recorded values")
	}
}
