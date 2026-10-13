package telemetry

import (
	"testing"
)

func TestSampler(t *testing.T) {
	sampler := NewSampler(1000) // 10%

	// Test basic sampling
	sampledCount := 0
	for i := 0; i < 10000; i++ {
		if sampler.ShouldSample() {
			sampledCount++
		}
	}

	if sampledCount < 800 || sampledCount > 1200 {
		t.Errorf("Expected ~1000 samples, got %d", sampledCount)
	}

	// Test error feedback
	initialRate := sampler.rate.Load()
	sampler.RecordError()
	if sampler.rate.Load() <= initialRate {
		t.Error("Sampling rate should increase after error")
	}

	total, _, errors := sampler.GetStats()
	if total != 10000 || errors != 1 {
		t.Errorf("Stats mismatch: total=%d, errors=%d", total, errors)
	}
}
