package reassembler

import (
	"errors"
)

type Reassembler struct {
	buffer *TLSBuffer
}

func NewReassembler() *Reassembler {
	return &Reassembler{buffer: NewTLSBuffer()}
}

func (r *Reassembler) Reassemble(chunk []byte) ([]byte, error) {
	if r.buffer.Len+len(chunk) > len(r.buffer.Data) {
		return nil, errors.New("buffer overflow")
	}
	copy(r.buffer.Data[r.buffer.Len:], chunk)
	r.buffer.Len += len(chunk)

	if r.buffer.Len < 5 {
		return nil, nil
	}

	recordLen := int(r.buffer.Data[3])<<8 | int(r.buffer.Data[4])
	totalLen := 5 + recordLen

	if r.buffer.Len >= totalLen {
		result := make([]byte, totalLen)
		copy(result, r.buffer.Data[:totalLen])
		copy(r.buffer.Data, r.buffer.Data[totalLen:r.buffer.Len])
		r.buffer.Len -= totalLen
		return result, nil
	}

	return nil, nil
}
