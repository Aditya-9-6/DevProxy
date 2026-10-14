package ws

import (
	"sync"
)

type Assembler struct {
	mu        sync.Mutex
	fragments [][]byte
	totalLen  int
}

func NewAssembler() *Assembler {
	return &Assembler{}
}

func (a *Assembler) Add(f *Frame) []byte {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.fragments = append(a.fragments, f.Payload)
	a.totalLen += len(f.Payload)

	if f.Fin {
		full := make([]byte, a.totalLen)
		off := 0
		for _, p := range a.fragments {
			copy(full[off:], p)
			off += len(p)
		}
		a.fragments = nil
		a.totalLen = 0
		return full
	}
	return nil
}
