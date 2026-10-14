package ws

import (
	"sync/atomic"
)

type CacheLinePad [8]uint64

type Assembler struct {
	fragments [][]byte
	active    atomic.Uint64
	padding   CacheLinePad
}

func NewAssembler() *Assembler {
	return &Assembler{}
}

func (a *Assembler) Add(f *Frame) []byte {
	a.active.Add(1)
	a.fragments = append(a.fragments, f.Payload)
	if f.Fin {
		var total int
		for _, p := range a.fragments {
			total += len(p)
		}
		full := make([]byte, total)
		off := 0
		for _, p := range a.fragments {
			copy(full[off:], p)
			off += len(p)
		}
		a.fragments = nil
		return full
	}
	return nil
}
