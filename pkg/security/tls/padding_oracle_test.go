package tls

import (
	"sync"
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

func TestPaddingOracleDetectorConcurrent(t *testing.T) {
	t.Parallel()
	detector := NewPaddingOracleDetector()
	var wg sync.WaitGroup
	workers := 10
	iterations := 1000

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				detector.ObserveRecord(128)
			}
		}(i)
	}
	wg.Wait()

	expected := uint64(workers * iterations)
	if count := detector.GetRecordCount(128); count != expected {
		t.Errorf("Expected concurrent record count %d, got %d", expected, count)
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

	// Empty data edge case
	if ent := AnalyzePaddingEntropy(nil); ent != 0 {
		t.Errorf("Expected 0 entropy for nil data, got %f", ent)
	}
}

func BenchmarkAnalyzePaddingEntropy(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AnalyzePaddingEntropy(data)
	}
}
