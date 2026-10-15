package telemetry

import (
	"bytes"
	"sync"
	"testing"
)

func TestFlightRecorder(t *testing.T) {
	recorder := NewFlightRecorder()
	recorder.Record([]byte("event1"))
	recorder.Record([]byte("event2"))

	count := 0
	recorder.Flush(func(b []byte) {
		count++
	})

	if count != 2 {
		t.Errorf("expected 2 events, got %d", count)
	}
}

func TestFlightRecorder_LargePayload(t *testing.T) {
	recorder := NewFlightRecorder()
	largeData := make([]byte, 4096)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	recorder.Record(largeData)

	recorder.Flush(func(b []byte) {
		if !bytes.Equal(b, largeData) {
			t.Error("data mismatch in large payload")
		}
	})
}

func TestFlightRecorder_ConcurrentLoad(t *testing.T) {
	recorder := NewFlightRecorder()
	const goroutines = 20
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				recorder.Record([]byte("concurrent-event"))
				if j%50 == 0 {
					recorder.Flush(func(b []byte) {})
				}
			}
		}()
	}

	wg.Wait()
	recorder.Flush(func(b []byte) {})
}
