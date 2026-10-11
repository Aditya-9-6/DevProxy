package reassembler

import (
	"errors"
	"sync"
)

var (
	ErrBufferOverflow = errors.New("tls reassembler: buffer overflow")
	ErrInvalidHeader  = errors.New("tls reassembler: invalid header")
)

type Reassembler struct {
	mu     sync.Mutex
	buffer *TLSBuffer
}

func NewReassembler() *Reassembler {
	return &Reassembler{buffer: NewTLSBuffer()}
}

func (r *Reassembler) Reassemble(chunk []byte) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.buffer.Len+len(chunk) > len(r.buffer.Data) {
		return nil, ErrBufferOverflow
	}

	copy(r.buffer.Data[r.buffer.Len:], chunk)
	r.buffer.Len += len(chunk)

	if r.buffer.Len < 5 {
		return nil, nil
	}

	recordLen := int(r.buffer.Data[3])<<8 | int(r.buffer.Data[4])
	totalLen := 5 + recordLen

	if r.buffer.Len >= totalLen {
		result := r.buffer.Data[:totalLen]
		remaining := r.buffer.Len - totalLen
		if remaining > 0 {
			copy(r.buffer.Data, r.buffer.Data[totalLen:r.buffer.Len])
		}
		r.buffer.Len = remaining
		return result, nil
	}

	return nil, nil
}
