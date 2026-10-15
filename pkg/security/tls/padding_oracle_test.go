package tls

import (
	"testing"
)

func TestPaddingOracleDetector(t *testing.T) {
	detector := NewPaddingOracleDetector()
	detector.ObserveRecord(100)
	detector.ObserveRecord(100)
	detector.ObserveRecord(200)

	if detector.GetRecordCount(100) != 2 {
		t.Errorf("Expected 2, got %d", detector.GetRecordCount(100))
	}
	if detector.GetRecordCount(200) != 1 {
		t.Errorf("Expected 1, got %d", detector.GetRecordCount(200))
	}
}

func TestEntropyAnalyzer(t *testing.T) {
	// High entropy (random)
	randomData := []byte{0x01, 0xFF, 0xAA, 0x55}
	entropy := AnalyzePaddingEntropy(randomData)
	if entropy <= 0 {
		t.Errorf("Expected positive entropy, got %f", entropy)
	}

	// Low entropy (repetitive)
	lowEntropyData := []byte{0x00, 0x00, 0x00, 0x00}
	entropyLow := AnalyzePaddingEntropy(lowEntropyData)
	if entropyLow != 0 {
		t.Errorf("Expected 0 entropy for repetitive data, got %f", entropyLow)
	}
}
