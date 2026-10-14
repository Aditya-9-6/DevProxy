package stream

import (
	"sync/atomic"
)

// Sequencer handles out-of-order TCP segments using atomic sequence tracking.
type Sequencer struct {
	expectedSeq atomic.Uint64
	head        atomic.Pointer[SegmentBuffer]
}

func NewSequencer(startSeq uint64) *Sequencer {
	s := &Sequencer{}
	s.expectedSeq.Store(startSeq)
	return s
}

// Push adds a segment to the re-sequencer. Returns true if data is ready for consumption.
func (s *Sequencer) Push(seq uint64, data []byte) bool {
	newSeg := GetSegment(seq, data)

	for {
		oldHead := s.head.Load()
		newSeg.Next = oldHead
		if s.head.CompareAndSwap(oldHead, newSeg) {
			break
		}
	}
	return seq == s.expectedSeq.Load()
}

// PopReady returns the next contiguous segment if available.
func (s *Sequencer) PopReady() *SegmentBuffer {
	expected := s.expectedSeq.Load()

	for {
		curr := s.head.Load()
		var prev *SegmentBuffer
		for curr != nil {
			if curr.Sequence == expected {
				if prev == nil {
					if s.head.CompareAndSwap(curr, curr.Next) {
						s.expectedSeq.Add(1)
						return curr
					}
				} else {
					prev.Next = curr.Next
					s.expectedSeq.Add(1)
					return curr
				}
			}
			prev = curr
			curr = curr.Next
		}
		return nil
	}
}
