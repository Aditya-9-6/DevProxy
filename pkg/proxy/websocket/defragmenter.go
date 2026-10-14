package websocket

import (
	"errors"
)

var ErrBufferOverflow = errors.New("websocket: fragment payload exceeds buffer capacity")

const MaxBufferSize = 10 * 1024 * 1024 // 10MB limit

type Defragmenter struct {
	buffer []byte
	offset int
}

func (d *Defragmenter) Write(p []byte) (int, error) {
	if d.buffer == nil {
		d.buffer = GetBuffer()
	}

	if d.offset+len(p) > MaxBufferSize {
		return 0, ErrBufferOverflow
	}

	if d.offset+len(p) > len(d.buffer) {
		newSize := len(d.buffer) * 2
		for d.offset+len(p) > newSize {
			newSize *= 2
		}
		if newSize > MaxBufferSize {
			newSize = MaxBufferSize
		}
		newBuf := make([]byte, newSize)
		copy(newBuf, d.buffer[:d.offset])
		PutBuffer(d.buffer)
		d.buffer = newBuf
	}

	copy(d.buffer[d.offset:], p)
	d.offset += len(p)
	return len(p), nil
}

func (d *Defragmenter) Reset() {
	if d.buffer != nil {
		PutBuffer(d.buffer)
		d.buffer = nil
	}
	d.offset = 0
}

func (d *Defragmenter) Payload() []byte {
	if d.buffer == nil {
		return nil
	}
	return d.buffer[:d.offset]
}
