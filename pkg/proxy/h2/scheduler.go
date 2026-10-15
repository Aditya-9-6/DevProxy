package h2

import (
	"container/heap"
	"sync"
)

// StreamWeight tracks the relative priority of an HTTP/2 stream.
// This struct is managed by the Scheduler and protected by its mutex.
type StreamWeight struct {
	Weight uint64
	index  int
}

// Scheduler manages fair-share bandwidth allocation for HTTP/2 streams using a Max-Heap.
// This implementation uses a RWMutex for thread-safe access to the underlying heap.
// Cache-line padding is included to prevent false sharing on the mutex.
type Scheduler struct {
	_       [56]byte
	mu      sync.RWMutex
	streams streamHeap
	_       [56]byte
}

type streamHeap []*StreamWeight

func (h streamHeap) Len() int { return len(h) }
func (h streamHeap) Less(i, j int) bool {
	return h[i].Weight > h[j].Weight
}
func (h streamHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *streamHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*StreamWeight)
	item.index = n
	*h = append(*h, item)
}
func (h *streamHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	item.index = -1
	*h = old[0 : n-1]
	return item
}

// NewScheduler initializes a new scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		streams: make(streamHeap, 0),
	}
}

// AddStream registers a new stream with an initial weight.
func (s *Scheduler) AddStream(initialWeight uint64) *StreamWeight {
	sw := &StreamWeight{Weight: initialWeight}
	s.mu.Lock()
	heap.Push(&s.streams, sw)
	s.mu.Unlock()
	return sw
}

// UpdateWeight updates the stream weight and re-heaps the scheduler.
func (sw *StreamWeight) UpdateWeight(s *Scheduler, newWeight uint64) {
	s.mu.Lock()
	sw.Weight = newWeight
	if sw.index >= 0 {
		heap.Fix(&s.streams, sw.index)
	}
	s.mu.Unlock()
}

// SelectStream returns the index of the highest weight stream in O(1).
func (s *Scheduler) SelectStream() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.streams) == 0 {
		return -1
	}
	return s.streams[0].index
}
