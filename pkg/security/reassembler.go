package security

import (
	"container/heap"
	"sync"
)

// Segment represents a TCP payload chunk.
type Segment struct {
	Seq  uint64
	Data []byte
}

// SegmentHeap implements heap.Interface for ordering segments by sequence number.
type SegmentHeap []*Segment

func (h SegmentHeap) Len() int            { return len(h) }
func (h SegmentHeap) Less(i, j int) bool  { return h[i].Seq < h[j].Seq }
func (h SegmentHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *SegmentHeap) Push(x interface{}) { *h = append(*h, x.(*Segment)) }
func (h *SegmentHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// StreamReassembler manages out-of-order TCP segments.
type StreamReassembler struct {
	mu         sync.Mutex
	nextSeq    uint64
	segments   SegmentHeap
	bufferPool *sync.Pool
}

func NewStreamReassembler(initialSeq uint64) *StreamReassembler {
	return &StreamReassembler{
		nextSeq:  initialSeq,
		segments: make(SegmentHeap, 0),
		bufferPool: &sync.Pool{
			New: func() interface{} { return make([]byte, 4096) },
		},
	}
}

func (r *StreamReassembler) Push(seq uint64, data []byte) [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	buf := r.bufferPool.Get().([]byte)
	copy(buf, data)
	heap.Push(&r.segments, &Segment{Seq: seq, Data: buf[:len(data)]})

	var ready [][]byte
	for r.segments.Len() > 0 && r.segments[0].Seq <= r.nextSeq {
		seg := heap.Pop(&r.segments).(*Segment)
		if seg.Seq+uint64(len(seg.Data)) > r.nextSeq {
			ready = append(ready, seg.Data)
			r.nextSeq = seg.Seq + uint64(len(seg.Data))
		}
		r.bufferPool.Put(seg.Data[:cap(seg.Data)])
	}
	return ready
}
