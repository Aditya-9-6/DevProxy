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

// StreamReassembler manages out-of-order TCP segments using a mutex.
type StreamReassembler struct {
	mu       sync.Mutex
	nextSeq  uint64
	segments SegmentHeap
	pool4k   *sync.Pool
	pool64k  *sync.Pool
	maxSegs  int
}

func NewStreamReassembler(initialSeq uint64) *StreamReassembler {
	return &StreamReassembler{
		nextSeq:  initialSeq,
		segments: make(SegmentHeap, 0),
		maxSegs:  1000,
		pool4k:   &sync.Pool{New: func() interface{} { return make([]byte, 4096) }},
		pool64k:  &sync.Pool{New: func() interface{} { return make([]byte, 65536) }},
	}
}

func (r *StreamReassembler) getBuffer(size int) []byte {
	if size <= 4096 {
		return r.pool4k.Get().([]byte)[:size]
	} else if size <= 65536 {
		return r.pool64k.Get().([]byte)[:size]
	}
	return make([]byte, size)
}

func (r *StreamReassembler) putBuffer(buf []byte) {
	cap := cap(buf)
	if cap == 4096 {
		r.pool4k.Put(buf[:cap])
	} else if cap == 65536 {
		r.pool64k.Put(buf[:cap])
	}
}

// Push adds a segment and invokes the callback for each ordered chunk.
func (r *StreamReassembler) Push(seq uint64, data []byte, onData func([]byte)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.segments.Len() >= r.maxSegs {
		return
	}

	if seq < r.nextSeq {
		offset := r.nextSeq - seq
		if offset >= uint64(len(data)) {
			return
		}
		data = data[offset:]
		seq = r.nextSeq
	}

	buf := r.getBuffer(len(data))
	copy(buf, data)
	heap.Push(&r.segments, &Segment{Seq: seq, Data: buf})

	for r.segments.Len() > 0 && r.segments[0].Seq <= r.nextSeq {
		seg := heap.Pop(&r.segments).(*Segment)
		if seg.Seq+uint64(len(seg.Data)) > r.nextSeq {
			onData(seg.Data)
			r.nextSeq = seg.Seq + uint64(len(seg.Data))
		}
		r.putBuffer(seg.Data)
	}
}
