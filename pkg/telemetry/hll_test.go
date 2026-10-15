package telemetry

import (
	"fmt"
	"sync"
	"testing"
)

func TestHyperLogLogEstimator(t *testing.T) {
	t.Parallel()

	t.Run("Basic", func(t *testing.T) {
		hll := NewHyperLogLogEstimator()
		for _, ip := range []string{"192.168.1.1", "192.168.1.2", "10.0.0.1", "172.16.0.5", "8.8.8.8"} {
			hll.AddString(ip)
		}
		estimate := hll.Estimate()
		if estimate < 3 || estimate > 8 {
			t.Errorf("Expected cardinality around 5, got %d", estimate)
		}
	})

	t.Run("Scale", func(t *testing.T) {
		hll := NewHyperLogLogEstimator()
		count := 1000
		for i := 0; i < count; i++ {
			hll.AddString(fmt.Sprintf("10.100.%d.%d", i/256, i%256))
		}
		estimate := hll.Estimate()
		if estimate < uint64(float64(count)*0.85) || estimate > uint64(float64(count)*1.15) {
			t.Errorf("Scale test estimate %d out of range", estimate)
		}
	})

	t.Run("Concurrency", func(t *testing.T) {
		hll := NewHyperLogLogEstimator()
		var wg sync.WaitGroup
		for w := 0; w < 10; w++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					hll.AddString(fmt.Sprintf("192.168.%d.%d", id, i))
				}
			}(w)
		}
		wg.Wait()
		estimate := hll.Estimate()
		if estimate < 800 || estimate > 1200 {
			t.Errorf("Concurrent estimate %d out of range", estimate)
		}
	})

	t.Run("Reset", func(t *testing.T) {
		hll := NewHyperLogLogEstimator()
		hll.AddString("1.2.3.4")
		hll.Reset()
		if hll.Estimate() != 0 {
			t.Errorf("Expected 0 after reset")
		}
	})
}
