package websocket

type Defragmenter struct {
	buffer []byte
	offset int
}

func (d *Defragmenter) Write(p []byte) (int, error) {
	if d.buffer == nil {
		d.buffer = GetBuffer()
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
	return d.buffer[:d.offset]
}
