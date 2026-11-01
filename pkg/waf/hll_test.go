package waf

import (
	"testing"
)

func TestHyperLogLog(t *testing.T) {
	hll := NewHyperLogLog(10)
	hll.Add(12345)
	hll.Add(67890)
	
	count := hll.Count()
	if count == 0 {
		t.Errorf("expected non-zero count, got %d", count)
	}
}

func TestHyperLogLogEstimate(t *testing.T) {
	hll := NewHyperLogLog(12)
	for i := uint64(0); i < 1000; i++ {
		hll.Add(i * 9999)
	}
	
	count := hll.Count()
	var val float64 = float64(count)
	if val < 500 || val > 1500 {
		t.Errorf("expected count around 1000, got %d", count)
	}
}
