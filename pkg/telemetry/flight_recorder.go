package telemetry

import (
	"sync/atomic"
)

// FlightRecorder implements a lock-free circular trace buffer.
type FlightRecorder struct {
	buffer [1024]atomic.Pointer[[]byte]
	cursor atomic.Uint32
}

func NewFlightRecorder() *FlightRecorder {
	return &FlightRecorder{}
}

// Record captures a trace event into the circular buffer using atomic swaps.
func (f *FlightRecorder) Record(data []byte) {
	idx := f.cursor.Add(1) % 1024
	buf := BufferPool.Get().([]byte)
	if len(buf) < len(data) {
		buf = make([]byte, len(data))
	}
	copy(buf, data)
	finalBuf := buf[:len(data)]

	// Atomically swap the buffer, returning the old one to the pool if it exists
	old := f.buffer[idx].Swap(&finalBuf)
	if old != nil {
		BufferPool.Put(*old)
	}
}

// Flush processes all captured events via a callback to avoid allocations.
func (f *FlightRecorder) Flush(fn func([]byte)) {
	for i := 0; i < 1024; i++ {
		old := f.buffer[i].Swap(nil)
		if old != nil {
			fn(*old)
			BufferPool.Put(*old)
		}
	}
}
