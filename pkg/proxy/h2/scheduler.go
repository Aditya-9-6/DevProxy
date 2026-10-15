package h2

import (
	"sync"
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to 64 bytes.
type CacheLinePad [8]uint64

// StreamWeight tracks the relative priority of an HTTP/2 stream.
type StreamWeight struct {
	Weight CacheLinePad
}

// Scheduler manages fair-share bandwidth allocation for HTTP/2 streams.
type Scheduler struct {
	mu      sync.RWMutex
	streams []*StreamWeight
}

// NewScheduler initializes a new scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		streams: make([]*StreamWeight, 0),
	}
}

// AddStream registers a new stream with an initial weight.
func (s *Scheduler) AddStream(initialWeight uint64) *StreamWeight {
	sw := &StreamWeight{}
	sw.Weight[0] = initialWeight
	s.mu.Lock()
	s.streams = append(s.streams, sw)
	s.mu.Unlock()
	return sw
}

// UpdateWeight updates the stream weight using atomic store to ensure thread safety.
func (sw *StreamWeight) UpdateWeight(newWeight uint64) {
	atomic.StoreUint64(&sw.Weight[0], newWeight)
}

// SelectStream performs a weighted round-robin selection (simplified fair-share).
func (s *Scheduler) SelectStream() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.streams) == 0 {
		return -1
	}

	bestIdx := 0
	var maxW uint64
	for i, sw := range s.streams {
		w := atomic.LoadUint64(&sw.Weight[0])
		if w > maxW {
			maxW = w
			bestIdx = i
		}
	}
	return bestIdx
}
