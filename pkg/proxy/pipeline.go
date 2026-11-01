package proxy

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrPipelineFull   = errors.New("pipeline: queue is full (backpressure exceeded)")
	ErrPipelineClosed = errors.New("pipeline: queue is closed")
	ErrQueueEmpty     = errors.New("pipeline: queue is empty")
)

// PipelinedRequest represents a single multiplexed HTTP/1.1 frame or request item.
type PipelinedRequest struct {
	ID      uint64
	Payload []byte
	Context context.Context
}

// CacheLinePad prevents false sharing by padding struct fields across typical 64-byte CPU cache lines.
type CacheLinePad struct {
	_ [64]byte
}

// AtomicRingBuffer implements a lock-free atomic single-producer/single-consumer or multi-producer/multi-consumer
// ring buffer optimized for high-throughput zero-allocation HTTP/1.1 request pipelining.
type AtomicRingBuffer struct {
	_        CacheLinePad
	head     atomic.Uint64
	_        CacheLinePad
	tail     atomic.Uint64
	_        CacheLinePad
	capacity uint64
	mask     uint64
	buffer   []atomic.Pointer[PipelinedRequest]
}

// NewAtomicRingBuffer initializes a ring buffer with capacity rounded up to the nearest power of two.
func NewAtomicRingBuffer(capacity uint64) *AtomicRingBuffer {
	cap := uint64(1)
	for cap < capacity {
		cap <<= 1
	}
	
	buf := make([]atomic.Pointer[PipelinedRequest], cap)
	return &AtomicRingBuffer{
		capacity: cap,
		mask:     cap - 1,
		buffer:   buf,
	}
}

// Push adds a request to the ring buffer lock-free. Returns ErrPipelineFull if full.
func (rb *AtomicRingBuffer) Push(req *PipelinedRequest) bool {
	head := rb.head.Load()
	tail := rb.tail.Load()

	if tail-head >= rb.capacity {
		return false
	}

	if !rb.tail.CompareAndSwap(tail, tail+1) {
		return false
	}

	idx := tail & rb.mask
	rb.buffer[idx].Store(req)
	return true
}

// Pop removes and returns a request from the ring buffer lock-free. Returns nil if empty.
func (rb *AtomicRingBuffer) Pop() *PipelinedRequest {
	for {
		head := rb.head.Load()
		tail := rb.tail.Load()

		if head >= tail {
			return nil
		}

		idx := head & rb.mask
		req := rb.buffer[idx].Load()

		if rb.head.CompareAndSwap(head, head+1) {
			rb.buffer[idx].Store(nil)
			return req
		}
	}
}

// Len returns the current approximate number of items in the buffer.
func (rb *AtomicRingBuffer) Len() uint64 {
	head := rb.head.Load()
	tail := rb.tail.Load()
	if tail < head {
		return 0
	}
	return tail - head
}

// PipelineQueue manages zero-allocation HTTP/1.1 request pipelining with atomic backpressure limits.
type PipelineQueue struct {
	ring     *AtomicRingBuffer
	maxDepth uint32
	inflight atomic.Int32
	closed   atomic.Bool
	pool     sync.Pool
}

// NewPipelineQueue instantiates a pipelined request manager with a backpressure limit.
func NewPipelineQueue(capacity uint32) *PipelineQueue {
	return &PipelineQueue{
		ring:     NewAtomicRingBuffer(uint64(capacity)),
		maxDepth: capacity,
		pool: sync.Pool{
			New: func() any {
				return &PipelinedRequest{}
			},
		},
	}
}

// AcquireRequest retrieves a zero-allocation PipelinedRequest from the pool.
func (pq *PipelineQueue) AcquireRequest() *PipelinedRequest {
	return pq.pool.Get().(*PipelinedRequest)
}

// ReleaseRequest returns a PipelinedRequest to the pool.
func (pq *PipelineQueue) ReleaseRequest(req *PipelinedRequest) {
	if req == nil {
		return
	}
	req.ID = 0
	req.Payload = nil
	req.Context = nil
	pq.pool.Put(req)
}

// Enqueue adds a pipelined request if backpressure limits are respected.
func (pq *PipelineQueue) Enqueue(req *PipelinedRequest) error {
	if pq.closed.Load() {
		return ErrPipelineClosed
	}

	// Check backpressure depth via inflight and ring capacity
	currentInflight := pq.inflight.Load()
	if currentInflight >= int32(pq.maxDepth) {
		return ErrPipelineFull
	}

	if !pq.ring.Push(req) {
		return ErrPipelineFull
	}

	pq.inflight.Add(1)
	return nil
}

// Dequeue extracts the next pipelined request for processing.
func (pq *PipelineQueue) Dequeue() (*PipelinedRequest, error) {
	if pq.closed.Load() && pq.ring.Len() == 0 {
		return nil, ErrPipelineClosed
	}

	req := pq.ring.Pop()
	if req == nil {
		return nil, ErrQueueEmpty
	}

	pq.inflight.Add(-1)
	return req, nil
}

// Close shuts down the pipeline queue.
func (pq *PipelineQueue) Close() {
	pq.closed.Store(true)
}

// Inflight returns the current count of active requests in the pipeline.
func (pq *PipelineQueue) Inflight() int32 {
	return pq.inflight.Load()
}
