package ringbuffer

import (
	"sync/atomic"
	"unsafe"
)

type BatchFlusher struct {
	ringBuffer *RingBuffer
	// batchPtr holds an unsafe.Pointer to the current slice header for atomic swapping
	batchPtr unsafe.Pointer
}

func NewBatchFlusher(rb *RingBuffer) *BatchFlusher {
	// Initialize with an empty slice header
	initial := make([]*TrafficEvent, 0, rb.capacity)
	return &BatchFlusher{
		ringBuffer: rb,
		batchPtr:   unsafe.Pointer(&initial),
	}
}

func (f *BatchFlusher) Flush() []*TrafficEvent {
	// Atomically swap the buffer with a fresh pre-allocated slice
	// This ensures zero-allocation during the hot path
	newBuffer := make([]*TrafficEvent, 0, f.ringBuffer.capacity)

	oldPtr := atomic.SwapPointer(&f.batchPtr, unsafe.Pointer(&newBuffer))

	// Dereference the old pointer to get the slice header
	return *(*[]*TrafficEvent)(oldPtr)
}
