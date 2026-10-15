package tdigest

import (
	"sync"
	"testing"
)

func TestDigest_AddAndQuery(t *testing.T) {
	d := NewDigest(100)
	for i := uint64(1); i <= 1000; i++ {
		d.Add(i)
	}

	q50 := d.GetQuantile(0.5)
	if q50 == 0 {
		t.Errorf("Expected non-zero quantile, got 0")
	}
}

func TestDigest_TableDrivenQuantiles(t *testing.T) {
	tests := []struct {
		name      string
		values    []uint64
		quantile  float64
		expectMin uint64
		expectMax uint64
	}{
		{
			name:      "Empty Digest",
			values:    []uint64{},
			quantile:  0.5,
			expectMin: 0,
			expectMax: 0,
		},
		{
			name:      "Single Value",
			values:    []uint64{42},
			quantile:  0.5,
			expectMin: 42,
			expectMax: 42,
		},
		{
			name:      "Boundary Zero Quantile",
			values:    []uint64{10, 20, 30, 40, 50},
			quantile:  0.0,
			expectMin: 10,
			expectMax: 10,
		},
		{
			name:      "Boundary One Quantile",
			values:    []uint64{10, 20, 30, 40, 50},
			quantile:  1.0,
			expectMin: 50,
			expectMax: 50,
		},
		{
			name:      "Median Quantile",
			values:    []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			quantile:  0.5,
			expectMin: 4,
			expectMax: 6,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := NewDigest(50)
			for _, v := range tt.values {
				d.Add(v)
			}
			got := d.GetQuantile(tt.quantile)
			if tt.expectMin != 0 && got < tt.expectMin {
				t.Errorf("GetQuantile(%v) = %v, expected >= %v", tt.quantile, got, tt.expectMin)
			}
			if tt.expectMax != 0 && got > tt.expectMax {
				t.Errorf("GetQuantile(%v) = %v, expected <= %v", tt.quantile, got, tt.expectMax)
			}
		})
	}
}

func TestDigest_ConcurrentParallel(t *testing.T) {
	t.Parallel()
	d := NewDigest(200)
	var wg sync.WaitGroup

	workers := 10
	itemsPerWorker := 100

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(base uint64) {
			defer wg.Done()
			for i := 0; i < itemsPerWorker; i++ {
				d.Add(base + uint64(i))
			}
		}(uint64(w * itemsPerWorker))
	}

	wg.Wait()

	p50 := d.GetQuantile(0.5)
	if p50 == 0 {
		t.Errorf("Expected valid p50 quantile after concurrent additions, got 0")
	}
}
