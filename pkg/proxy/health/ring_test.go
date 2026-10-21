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
			for j := 0; j < 100; j++ {
				ring.Record(j%2 == 0, uint64(j*10))
			}
		}()
	}

	wg.Wait()

	rate := ring.SuccessRate()
	if rate < 0 || rate > 1 {
		t.Errorf("SuccessRate() out of range: %v", rate)
	}
}
