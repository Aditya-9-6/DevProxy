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
			New: func() interface{} { return &Packet{} },
		},
	}
}

func (jb *JitterBuffer) Push(seq uint64, data []byte) {
	idx := seq % BufferSize
	p := jb.pool.Get().(*Packet)
	p.Sequence = seq
	p.Data = data
	jb.buffer[idx].Store(p)
}

func (jb *JitterBuffer) Pop() *Packet {
	next := jb.nextSeq.Load()
	idx := next % BufferSize

	p := jb.buffer[idx].Swap(nil)
	if p == nil {
		return nil
	}

	jb.nextSeq.Add(1)
	return p
}

func (jb *JitterBuffer) Recycle(p *Packet) {
	p.Data = nil
	jb.pool.Put(p)
}
