package waf

import (
	"testing"
)

func TestCookieSecurityEngine(t *testing.T) {
	engine := NewCookieSecurityEngine()

	// Test Entropy
	lowEntropy := engine.AnalyzeEntropy("aaaaaaaaaa")
	highEntropy := engine.AnalyzeEntropy("a1b2c3d4e5f6")

	if lowEntropy >= highEntropy {
		t.Errorf("Expected higher entropy for complex string, got %f vs %f", highEntropy, lowEntropy)
	}

	// Test Atomic Counters
	engine.RecordViolation()
	engine.RecordViolation()

	if count := engine.GetViolationCount(); count != 2 {
		t.Errorf("Expected 2 violations, got %d", count)
	}
}
