package telemetry

import (
	"testing"
)

func TestBaggageBuffer(t *testing.T) {
	buf := NewBaggageBuffer()

	// Test Push and Get
	buf.Push("trace-id", "12345")
	buf.Push("span-id", "67890")

	items := buf.Get()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	foundTrace := false
	for _, item := range items {
		if item.Key == "trace-id" && item.Value == "12345" {
			foundTrace = true
		}
	}

	if !foundTrace {
		t.Error("trace-id not found in baggage")
	}
}

func TestBaggageBufferOverflow(t *testing.T) {
	buf := NewBaggageBuffer()
	// Fill beyond capacity
	for i := 0; i < MaxBaggageItems+10; i++ {
		buf.Push("k", "v")
	}

	items := buf.Get()
	if len(items) > MaxBaggageItems {
		t.Errorf("buffer exceeded capacity: %d", len(items))
	}
}
