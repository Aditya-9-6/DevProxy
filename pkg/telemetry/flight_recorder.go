package telemetry

import (
	"sync"
	"sync/atomic"
)

// FlightRecorder implements a thread-safe circular trace buffer.
type FlightRecorder struct {
	mu     sync.Mutex
	buffer [1024][]byte
	cursor atomic.Uint32
}

func NewFlightRecorder() *FlightRecorder {
	return &FlightRecorder{}
}

// Record captures a trace event into the circular buffer.
func (f *FlightRecorder) Record(data []byte) {
	idx := f.cursor.Add(1) % 1024
	buf := BufferPool.Get().([]byte)
	if len(buf) < len(data) {
		// If data exceeds default pool size, reallocate or use appropriately
		buf = make([]byte, len(data))
	}
	copy(buf, data)

	f.mu.Lock()
	defer f.mu.Unlock()

	// If an existing buffer was in this slot, return it to the pool first
	if f.buffer[idx] != nil {
		BufferPool.Put(f.buffer[idx])
	}
	f.buffer[idx] = buf[:len(data)]
}

// Flush returns all captured events and clears the buffer, returning buffers to the pool.
func (f *FlightRecorder) Flush() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()

	var results [][]byte
	for i := 0; i < 1024; i++ {
		if f.buffer[i] != nil {
			results = append(results, f.buffer[i])
			f.buffer[i] = nil
		}
	}
	return results
}
