package ws

import (
	"errors"
	"sync"
)

// Assembler manages the reassembly of fragmented WebSocket frames.
// It uses a mutex to protect state and a pooled buffer for zero-allocation reassembly.
type Assembler struct {
	mu        sync.Mutex
	fragments [][]byte
	totalLen  int
	opcode    byte
}

func NewAssembler() *Assembler {
	return &Assembler{}
}

// Add processes a frame. If Fin is true, it returns the reassembled payload.
// The caller is responsible for calling PutBuffer on the returned slice.
func (a *Assembler) Add(f *Frame) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.fragments) == 0 {
		a.opcode = f.Opcode
	} else if f.Opcode != 0 && f.Opcode != 0x00 {
		return nil, errors.New("invalid opcode in continuation frame")
	}

	a.fragments = append(a.fragments, f.Payload)
	a.totalLen += len(f.Payload)

	if f.Fin {
		full := GetBuffer()[:a.totalLen]
		off := 0
		for _, p := range a.fragments {
			copy(full[off:], p)
			PutBuffer(p)
			off += len(p)
		}
		a.fragments = nil
		a.totalLen = 0
		return full, nil
	}
	return nil, nil
}
