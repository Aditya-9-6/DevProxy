package security

import (
	"testing"
)

func TestCalculateShannonEntropy(t *testing.T) {
	table := []struct {
		name   string
		input  []byte
		assert func(float64)
	}{
		{
			name:  "empty",
			input: []byte{},
			assert: func(e float64) {
				if e != 0.0 {
					t.Errorf("expected 0 entropy, got %f", e)
				}
			},
		},
		{
			name:  "uniform",
			input: []byte{0x01, 0x02, 0x03, 0x04},
			assert: func(e float64) {
				if e <= 0.0 {
					t.Errorf("expected positive entropy, got %f", e)
				}
			},
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			entropy := CalculateShannonEntropy(tc.input)
			tc.assert(entropy)
		})
	}
}

func TestCacheLinePaddedScore(t *testing.T) {
	var score CacheLinePaddedScore
	if score.Load() != 0 {
		t.Fatalf("initial score must be 0")
	}

	score.Store(42)
	if score.Load() != 42 {
		t.Fatalf("expected 42, got %d", score.Load())
	}

	score.Add(8)
	if score.Load() != 50 {
		t.Fatalf("expected 50, got %d", score.Load())
	}
}
