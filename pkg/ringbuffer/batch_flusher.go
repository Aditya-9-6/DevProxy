package ringbuffer

import (
	"unsafe"
)

type BatchFlusher struct {
	ringBuffer *RingBuffer
	batchPtr   unsafe.Pointer
}

func NewBatchFlusher(rb *RingBuffer) *BatchFlusher {
	return &BatchFlusher{
		ringBuffer: rb,
	}
}

func (f *BatchFlusher) Flush() []*TrafficEvent {
	// Retrieve all current events from the ring buffer and reset it
	// This assumes the RingBuffer has a method to retrieve and clear its contents
	// or we access the underlying slice directly if exported.
	// Given the error, we implement the logic to drain the buffer safely.
	f.ringBuffer.mu.Lock()
	defer f.ringBuffer.mu.Unlock()

	items := f.ringBuffer.buffer
	f.ringBuffer.buffer = make([]*TrafficEvent, 0, cap(f.ringBuffer.buffer))
	return items
}
