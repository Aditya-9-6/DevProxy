package h3flow

import (
	"testing"
)

func TestTelemetryEngine(t *testing.T) {
	engine := NewTelemetryEngine()

	// Record some RTT values
	engine.RecordRTT(10)
	engine.RecordRTT(20)
	engine.RecordRTT(30)

	p50 := engine.GetPercentile(0.5)
	if p50 < 10 || p50 > 30 {
		t.Errorf("Expected p50 in range [10, 30], got %d", p50)
	}
}

func TestFlowControl(t *testing.T) {
	fc := NewFlowControl(100)

	if !fc.CanSend(50) {
		t.Error("Should be able to send 50 bytes with window 100")
	}

	fc.InFlightBytes.Add(60)
	if fc.CanSend(50) {
		t.Error("Should not be able to send 50 bytes when 60 in flight with window 100")
	}
}
