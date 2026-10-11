package security

import (
	"testing"
)

func TestEntropyAnalyzer(t *testing.T) {
	analyzer := NewEntropyAnalyzer()

	// Test low entropy (repetitive data)
	analyzer.Reset()
	analyzer.Add([]byte("AAAAAAAAAAAAAAAAAAAA"))
	entropyLow := analyzer.Calculate()
	if entropyLow > 0.1 {
		t.Errorf("Expected low entropy for repetitive data, got %f", entropyLow)
	}

	// Test high entropy (random-like data)
	analyzer.Reset()
	analyzer.Add([]byte("\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0A\x0B\x0C\x0D\x0E\x0F"))
	entropyHigh := analyzer.Calculate()
	if entropyHigh < 3.0 {
		t.Errorf("Expected high entropy for diverse data, got %f", entropyHigh)
	}
}
