package security

import (
	"container/heap"
	"sync"
)

type bufferWrapper struct {
	buf []byte
	cap int
}

type Segment struct {
	Seq  uint64
	Data []byte
	cap  int
}

var segmentPool = sync.Pool{New: func() interface{} { return &Segment{} }}

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
		pool4k:   &sync.Pool{New: func() interface{} { return &bufferWrapper{buf: make([]byte, 4096), cap: 4096} }},
		pool64k:  &sync.Pool{New: func() interface{} { return &bufferWrapper{buf: make([]byte, 65536), cap: 65536} }},
	}
}

func (r *StreamReassembler) getBuffer(size int) (buf []byte, cap int) {
	if size <= 4096 {
		w := r.pool4k.Get().(*bufferWrapper)
		return w.buf[:size], w.cap
	} else if size <= 65536 {
		w := r.pool64k.Get().(*bufferWrapper)
		return w.buf[:size], w.cap
	}
	return make([]byte, size), size
}

func (r *StreamReassembler) putBuffer(buf []byte, cap int) {
	if cap == 4096 {
		r.pool4k.Put(&bufferWrapper{buf: buf[:cap], cap: cap})
	} else if cap == 65536 {
		r.pool64k.Put(&bufferWrapper{buf: buf[:cap], cap: cap})
	}
}

func (r *StreamReassembler) Push(seq uint64, data []byte, onData func([]byte)) {
	var ready [][]byte
	r.mu.Lock()
	if r.segments.Len() < r.maxSegs {
		if seq < r.nextSeq {
			offset := r.nextSeq - seq
			if offset < uint64(len(data)) {
				data = data[offset:]
				seq = r.nextSeq
			}
		}
		buf, cap := r.getBuffer(len(data))
		copy(buf, data)
		seg := segmentPool.Get().(*Segment)
		seg.Seq, seg.Data, seg.cap = seq, buf, cap
		heap.Push(&r.segments, seg)
		for r.segments.Len() > 0 && r.segments[0].Seq <= r.nextSeq {
			s := heap.Pop(&r.segments).(*Segment)
			if s.Seq+uint64(len(s.Data)) > r.nextSeq {
				ready = append(ready, s.Data)
				r.nextSeq = s.Seq + uint64(len(s.Data))
			}
			r.putBuffer(s.Data, s.cap)
			segmentPool.Put(s)
		}
	}
	r.mu.Unlock()
	for _, b := range ready {
		onData(b)
	}
}
