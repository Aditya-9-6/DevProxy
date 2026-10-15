package telemetry

import (
	"sync/atomic"
)

// FlightRecorder implements a lock-free circular trace buffer.
type FlightRecorder struct {
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
	copy(buf, data)
	f.buffer[idx] = buf
}

// Flush returns all captured events and clears the buffer.
func (f *FlightRecorder) Flush() [][]byte {
	var results [][]byte
	for i := 0; i < 1024; i++ {
		if f.buffer[i] != nil {
			results = append(results, f.buffer[i])
			f.buffer[i] = nil
		}
	}
	return results
}
