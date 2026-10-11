package ws

import (
	"io"
)

// FrameTransformer handles streaming WebSocket frame transformation.
type FrameTransformer struct {
	maskKey [4]byte
}

// NewFrameTransformer creates a transformer with a specific masking key.
func NewFrameTransformer(key []byte) *FrameTransformer {
	ft := &FrameTransformer{}
	copy(ft.maskKey[:], key)
	return ft
}

// Transform reads from src, applies XOR masking, and writes to dst.
func (ft *FrameTransformer) Transform(dst io.Writer, src io.Reader) (int64, error) {
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			MaskXOR(buf[:n], ft.maskKey[:])
			wn, werr := dst.Write(buf[:n])
			total += int64(wn)
			if werr != nil {
				return total, werr
			}
		}
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}
