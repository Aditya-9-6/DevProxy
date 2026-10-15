package h2

import (
	"container/heap"
	"sync"
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to 64 bytes.
type CacheLinePad [8]uint64

// StreamWeight tracks the relative priority of an HTTP/2 stream.
type StreamWeight struct {
	Weight CacheLinePad
	index  int
}

// Scheduler manages fair-share bandwidth allocation for HTTP/2 streams using a Max-Heap.
// This implementation uses a RWMutex for thread-safe access to the underlying heap.
type Scheduler struct {
	mu      sync.RWMutex
	streams streamHeap
}

type streamHeap []*StreamWeight

func (h streamHeap) Len() int { return len(h) }
func (h streamHeap) Less(i, j int) bool {
	return atomic.LoadUint64(&h[i].Weight[0]) > atomic.LoadUint64(&h[j].Weight[0])
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
	sw := &StreamWeight{}
	sw.Weight[0] = initialWeight
	s.mu.Lock()
	heap.Push(&s.streams, sw)
	s.mu.Unlock()
	return sw
}

// UpdateWeight updates the stream weight and re-heaps the scheduler.
func (sw *StreamWeight) UpdateWeight(s *Scheduler, newWeight uint64) {
	atomic.StoreUint64(&sw.Weight[0], newWeight)
	s.mu.Lock()
	if sw.index >= 0 {
		heap.Fix(&s.streams, sw.index)
	}
	s.mu.Unlock()
}

// SelectStream returns the index of the highest weight stream in O(log N).
func (s *Scheduler) SelectStream() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.streams) == 0 {
		return -1
	}
	return s.streams[0].index
}
