package ringbuffer

import (
	"sync"
)

// BatchFlusher provides a thread-safe way to collect and flush events from a RingBuffer.
// It uses a sync.Pool to reuse slice memory, avoiding allocations.
type BatchFlusher struct {
	rb     *RingBuffer
	mu     sync.Mutex
	buffer []*TrafficEvent
	pool   sync.Pool
}

// NewBatchFlusher creates a new BatchFlusher linked to the given RingBuffer.
func NewBatchFlusher(rb *RingBuffer) *BatchFlusher {
	return &BatchFlusher{
		rb:     rb,
		buffer: make([]*TrafficEvent, 0, rb.capacity),
		pool: sync.Pool{
			New: func() interface{} {
				return make([]*TrafficEvent, 0, rb.capacity)
			},
		},
	}
}

// Flush drains available events from the RingBuffer into the current batch and swaps buffers.
func (f *BatchFlusher) Flush() []*TrafficEvent {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Drain available items from the ring buffer
	if f.rb != nil {
		for {
			event := f.rb.TryPop()
			if event == nil {
				break
			}
			f.buffer = append(f.buffer, event)
		}
	}

	batch := f.buffer
	f.buffer = f.pool.Get().([]*TrafficEvent)

	return batch
}

// Recycle returns a used slice to the pool for future reuse.
func (f *BatchFlusher) Recycle(batch []*TrafficEvent) {
	if batch == nil {
		return
	}
	batch = batch[:0]
	f.pool.Put(batch)
}
