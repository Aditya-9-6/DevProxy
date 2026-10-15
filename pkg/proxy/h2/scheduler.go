package h2

import (
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
	// Using a slice of pointers to padded structs to ensure cache-line alignment
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
	s.streams = append(s.streams, sw)
	return sw
}

// UpdateWeight updates the stream weight using atomic CAS to ensure thread safety without mutexes.
func (sw *StreamWeight) UpdateWeight(newWeight uint64) {
	for {
		old := atomic.LoadUint64(&sw.Weight[0])
		if atomic.CompareAndSwapUint64(&sw.Weight[0], old, newWeight) {
			break
		}
	}
}

// SelectStream performs a weighted round-robin selection (simplified fair-share).
func (s *Scheduler) SelectStream() int {
	var totalWeight uint64
	for _, sw := range s.streams {
		totalWeight += atomic.LoadUint64(&sw.Weight[0])
	}

	if totalWeight == 0 {
		return 0
	}

	// In a production implementation, this would use a deterministic selection algorithm
	// based on the atomic weights. For this implementation, we return the index of the highest weight.
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
