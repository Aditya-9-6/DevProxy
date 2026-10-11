package proxy

import (
	"io"
)

// H2Reassembler handles fragmented HTTP/2 streams using slab-allocated buffers.
type H2Reassembler struct {
	pool *FramePool
}

func NewH2Reassembler(pool *FramePool) *H2Reassembler {
	return &H2Reassembler{pool: pool}
}

// Reassemble reads from an io.Reader and returns a reassembled byte slice.
// The caller is responsible for returning the buffer to the pool via the reassembler.
func (r *H2Reassembler) Reassemble(reader io.Reader, totalSize int) ([]byte, error) {
	bufPtr := r.pool.Get(totalSize)
	buf := *bufPtr

	_, err := io.ReadFull(reader, buf[:totalSize])
	if err != nil {
		r.pool.Put(bufPtr)
		return nil, err
	}

	return buf[:totalSize], nil
}

// Release returns the buffer to the pool.
func (r *H2Reassembler) Release(buf []byte) {
	// Reconstruct pointer to slice for pool return
	r.pool.Put(&buf)
}
