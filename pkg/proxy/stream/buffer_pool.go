package stream

import "sync"

// SegmentBuffer represents a re-usable buffer for TCP segments.
type SegmentBuffer struct {
	Data     []byte
	Sequence uint64
	Next     *SegmentBuffer
}

var segmentPool = sync.Pool{
	New: func() interface{} {
		return &SegmentBuffer{
			Data: make([]byte, 4096),
		}
	},
}

func GetSegment(seq uint64, data []byte) *SegmentBuffer {
	s := segmentPool.Get().(*SegmentBuffer)
	s.Sequence = seq
	copy(s.Data, data)
	return s
}

func PutSegment(s *SegmentBuffer) {
	segmentPool.Put(s)
}
