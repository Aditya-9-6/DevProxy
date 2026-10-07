package telemetry

import (
	"context"
	"testing"
)

func TestExporterInitialization(t *testing.T) {
	ctx := context.Background()

	// Should successfully initialize with a dummy endpoint
	exporter, err := NewExporter(ctx, "localhost:4317")
	if err != nil {
		t.Fatalf("Failed to initialize exporter: %v", err)
	}

	if exporter.Tracer() == nil {
		t.Error("Tracer should not be nil")
	}

	if exporter.Meter() == nil {
		t.Error("Meter should not be nil")
	}

	if exporter.Propagator() == nil {
		t.Error("Propagator should not be nil")
	}

	// Should not crash on shutdown. Connection errors are expected in tests since there's no backend running
	// so we just log it rather than failing the test.
	err = exporter.Shutdown(ctx)
	if err != nil {
		t.Logf("Shutdown returned an error as expected without a backend: %v", err)
	}
}

func TestSpanCreation(t *testing.T) {
	ctx := context.Background()

	exporter, err := NewExporter(ctx, "localhost:4317")
	if err != nil {
		t.Fatalf("Failed to initialize exporter: %v", err)
	}
	defer exporter.Shutdown(ctx)

	// Test trace creation
	_, span := exporter.Tracer().Start(ctx, "test-span")
	if span == nil {
		t.Fatal("Failed to start span")
	}

	// Add an event just to ensure it doesn't panic
	span.AddEvent("test-event")
	span.End()
}
