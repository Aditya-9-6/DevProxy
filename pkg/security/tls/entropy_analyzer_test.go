package tls

import (
	"math"
	"testing"
)

func TestAnalyzePaddingEntropy(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected float64 // Approximate expected entropy
		tolerance float64
	}{
		{
			name:     "Empty slice",
			data:     []byte{},
			expected: 0.0,
			tolerance: 0.0,
		},
		{
			name:     "Single byte",
			data:     []byte{0x00},
			expected: 0.0,
			tolerance: 0.0,
		},
		{
			name:     "All same bytes",
			data:     []byte{0x42, 0x42, 0x42, 0x42},
			expected: 0.0,
			tolerance: 0.0,
		},
		{
			name:     "Two different bytes",
			data:     []byte{0x01, 0x02, 0x01, 0x02},
			expected: 1.0, // log2(2) = 1
			tolerance: 0.001,
		},
		{
			name:     "Four different bytes",
			data:     []byte{0x01, 0x02, 0x03, 0x04},
			expected: 2.0, // log2(4) = 2
			tolerance: 0.001,
		},
		{
			name:     "Random-like bytes",
			data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			expected: 3.0, // log2(8) = 3
			tolerance: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AnalyzePaddingEntropy(tt.data)
			if math.Abs(got-tt.expected) > tt.tolerance {
				t.Errorf("AnalyzePaddingEntropy() = %v, want %v (tolerance %v)", got, tt.expected, tt.tolerance)
			}
		})
	}
}
