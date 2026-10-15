package telemetry

import (
	"testing"
)

func TestFlightRecorder(t *testing.T) {
	recorder := NewFlightRecorder()
	recorder.Record([]byte("event1"))
	recorder.Record([]byte("event2"))

	flushed := recorder.Flush()
	if len(flushed) != 2 {
		t.Errorf("expected 2 events, got %d", len(flushed))
	}
}
