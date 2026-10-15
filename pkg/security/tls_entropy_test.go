package security

import (
	"testing"
)

func TestShannonEntropy(t *testing.T) {
	data := []uint16{1, 2, 3, 4, 1, 2, 3, 4}
	entropy := CalculateShannonEntropy(data)
	if entropy < 1.99 {
		t.Errorf("Expected entropy near 2.0, got %f", entropy)
	}
}

func TestEntropyScoreAtomic(t *testing.T) {
	es := &EntropyScore{}
	es.UpdateScore(100)
	if es.GetScore() != 100 {
		t.Errorf("Atomic store/load failed")
	}
}
