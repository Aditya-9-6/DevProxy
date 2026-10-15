package telemetry

import (
	"sync"
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

	// Ensure flushed buffers are safely recycled (return them manually or via pool helper if needed)
	for _, buf := range flushed {
		BufferPool.Put(buf)
	}
}

func TestFlightRecorder_ConcurrentLoad(t *testing.T) {
	recorder := NewFlightRecorder()
	const goroutines = 20
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				recorder.Record([]byte("concurrent-event"))
				if j%50 == 0 {
					flushed := recorder.Flush()
					for _, buf := range flushed {
						BufferPool.Put(buf)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Final flush to clear remaining items
	flushed := recorder.Flush()
	for _, buf := range flushed {
		BufferPool.Put(buf)
	}
}
