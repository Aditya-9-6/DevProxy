package replay

import (
	"bytes"
	"io"
)

type ReplayBuffer struct {
	buf      *bytes.Buffer
	capacity int
}

func NewReplayBuffer(capacity int) *ReplayBuffer {
	return &ReplayBuffer{
		buf:      bytes.NewBuffer(make([]byte, 0, capacity)),
		capacity: capacity,
	}
}

func (r *ReplayBuffer) Write(p []byte) (int, error) {
	return r.buf.Write(p)
}

func (r *ReplayBuffer) Read(p []byte) (int, error) {
	return r.buf.Read(p)
}

func (r *ReplayBuffer) Bytes() []byte {
	return r.buf.Bytes()
}

func (r *ReplayBuffer) Reader() io.Reader {
	return bytes.NewReader(r.buf.Bytes())
}

func (r *ReplayBuffer) Reset() {
	r.buf.Reset()
}
