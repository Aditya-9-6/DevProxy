package security

import (
	"sync"
	"testing"
	"time"
)

func TestContinuationDetector_Basic(t *testing.T) {
	d := NewContinuationDetector()
	for i := 0; i < 101; i++ {
		flood := d.RecordFrame()
		if i == 100 && !flood {
			t.Errorf("expected flood detection at 101 frames")
		}
	}
}

func TestContinuationDetector_Concurrency(t *testing.T) {
	d := NewContinuationDetector()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				d.RecordFrame()
			}
		}()
	}
	wg.Wait()
}

func TestContinuationDetector_TemporalReset(t *testing.T) {
	d := NewContinuationDetector()
	// Fill window
	for i := 0; i < 101; i++ {
		d.RecordFrame()
	}

	// Simulate time passing to clear buckets
	time.Sleep(time.Second * 11)

	// Should not be flooding anymore
	if d.RecordFrame() {
		t.Errorf("expected detector to reset after time window")
	}
}
