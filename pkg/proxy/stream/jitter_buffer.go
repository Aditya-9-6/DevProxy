package stream

import (
	"sync"
	"sync/atomic"
)

const BufferSize = 1024

type Packet struct {
	Sequence uint64
	Data     []byte
}

type JitterBuffer struct {
	_       [64]byte // Cache line padding
	buffer  [BufferSize]atomic.Pointer[Packet]
	nextSeq atomic.Uint64
	pool    sync.Pool
}

func NewJitterBuffer() *JitterBuffer {
	return &JitterBuffer{
		pool: sync.Pool{
			New: func() interface{} {
				return &Packet{
					Data: make([]byte, 0, 256),
				}
			},
		},
	}
}

// Push validates the sequence against the current sliding window and stores the packet securely.
func (jb *JitterBuffer) Push(seq uint64, data []byte) bool {
	next := jb.nextSeq.Load()

	// Sliding window bounds check: reject packets too old or too far in the future
	if seq < next || seq >= next+BufferSize {
		return false
	}

	idx := seq % BufferSize
	p := jb.pool.Get().(*Packet)
	p.Sequence = seq

	// Ensure safe memory management by copying data to avoid external mutation
	if cap(p.Data) < len(data) {
		p.Data = make([]byte, len(data))
	} else {
		p.Data = p.Data[:len(data)]
	}
	copy(p.Data, data)

	// If slot already contains a packet, recycle it to avoid leaks
	if old := jb.buffer[idx].Swap(p); old != nil {
		jb.Recycle(old)
	}

	return true
}

// Pop retrieves the next expected packet in sequence, handling gaps safely.
func (jb *JitterBuffer) Pop() *Packet {
	next := jb.nextSeq.Load()
	idx := next % BufferSize

	p := jb.buffer[idx].Load()
	if p == nil {
		return nil
	}

	// Verify sequence match to prevent out-of-order delivery on wrap or gaps
	if p.Sequence != next {
		return nil
	}

	// Atomically clear the slot
	if !jb.buffer[idx].CompareAndSwap(p, nil) {
		return nil
	}

	jb.nextSeq.Add(1)
	return p
}

func (jb *JitterBuffer) Recycle(p *Packet) {
	if p == nil {
		return
	}
	p.Data = p.Data[:0]
	p.Sequence = 0
	jb.pool.Put(p)
}
