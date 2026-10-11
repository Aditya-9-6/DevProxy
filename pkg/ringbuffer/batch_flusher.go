package ringbuffer

import (
	"sync"
)

// BatchFlusher provides a thread-safe way to collect and flush events.
// It uses a sync.Pool to reuse slice memory, avoiding allocations.
type BatchFlusher struct {
	mu     sync.Mutex
	buffer []*TrafficEvent
	pool   sync.Pool
}

func NewBatchFlusher(rb *RingBuffer) *BatchFlusher {
	return &BatchFlusher{
		buffer: make([]*TrafficEvent, 0, rb.capacity),
		pool: sync.Pool{
			New: func() interface{} {
				return make([]*TrafficEvent, 0, rb.capacity)
			},
		},
	}
}

// Flush returns the current batch and resets the internal buffer.
func (f *BatchFlusher) Flush() []*TrafficEvent {
	f.mu.Lock()
	defer f.mu.Unlock()

	batch := f.buffer
	f.buffer = f.pool.Get().([]*TrafficEvent)

	return batch
}

// Recycle returns a used slice to the pool for future reuse.
func (f *BatchFlusher) Recycle(batch []*TrafficEvent) {
	batch = batch[:0]
	f.pool.Put(batch)
}
