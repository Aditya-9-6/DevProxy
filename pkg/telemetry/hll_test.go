package telemetry

import (
	"fmt"
	"sync"
	"testing"
)

func TestHyperLogLogEstimator_Basic(format *testing.T) {
	hll := NewHyperLogLogEstimator()

	// Add unique IP addresses
	ips := []string{
		"192.168.1.1",
		"192.168.1.2",
		"10.0.0.1",
		"172.16.0.5",
		"8.8.8.8",
	}

	for _, ip := range ips {
		hll.AddString(ip)
	}

	// Add duplicates
	hll.AddString("192.168.1.1")
	hll.AddString("8.8.8.8")

	estimate := hll.Estimate()
	// Expect approximate cardinality around 5
	if estimate < 3 || estimate > 8 {
		format.Errorf("Expected cardinality around 5, got %d", estimate)
	}
}

func TestHyperLogLogEstimator_Scale(t *testing.T) {
	hll := NewHyperLogLogEstimator()
	count := 1000

	for i := 0; i < count; i++ {
		ip := fmt.Sprintf("10.100.%d.%d", i/256, i%256)
		hll.AddString(ip)
	}

	estimate := hll.Estimate()
	// Allow 10% error margin for HLL
	lower := uint64(float64(count) * 0.85)
	upper := uint64(float64(count) * 1.15)

	if estimate < lower || estimate > upper {
		t.Errorf("Scale test cardinality estimate %d outside acceptable range [%d, %d]", estimate, lower, upper)
	}
}

func TestHyperLogLogEstimator_Concurrency(t *testing.T) {
	hll := NewHyperLogLogEstimator()
	var wg sync.WaitGroup
	workers := 10
	itemsPerWorker := 100

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < itemsPerWorker; i++ {
				ip := fmt.Sprintf("192.168.%d.%d", workerID, i)
				hll.AddString(ip)
			}
		}(w)
	}

	wg.Wait()
	estimate := hll.Estimate()

	// Total unique items = 10 * 100 = 1000
	lower := uint64(800)
	upper := uint64(1200)

	if estimate < lower || estimate > upper {
		t.Errorf("Concurrent HLL estimate %d outside expected bounds [%d, %d]", estimate, lower, upper)
	}
}

func TestHyperLogLogEstimator_Reset(t *testing.T) {
	hll := NewHyperLogLogEstimator()
	hll.AddString("1.2.3.4")
	if hll.Estimate() == 0 {
		t.Fatal("Expected non-zero estimate after adding item")
	}
	hll.Reset()
	if hll.Estimate() != 0 {
		t.Errorf("Expected zero estimate after reset, got %d", hll.Estimate())
	}
}
