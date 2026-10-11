package reassembler

import "sync"

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 65536)
	},
}

type TLSBuffer struct {
	Data []byte
	Len  int
}

func NewTLSBuffer() *TLSBuffer {
	return &TLSBuffer{Data: bufferPool.Get().([]byte), Len: 0}
}

func (b *TLSBuffer) Release() {
	bufferPool.Put(b.Data)
	b.Data = nil
}
