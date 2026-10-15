package tdigest

import (
	"testing"
)

func TestDigest_AddAndQuery(t *testing.T) {
	d := NewDigest(100)
	for i := uint64(1); i <= 100; i++ {
		d.Add(i)
	}

	q50 := d.GetQuantile(0.5)
	if q50 == 0 {
		t.Errorf("Expected non-zero quantile, got 0")
	}
}
