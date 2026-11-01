package http2

import (
	"io"
)

type FrameDecoder struct {
	maxFrameSize uint32
}

func NewFrameDecoder(maxFrameSize uint32) *FrameDecoder {
	return &FrameDecoder{
		maxFrameSize: maxFrameSize,
	}
}

func (d *FrameDecoder) DecodeHeader(r io.Reader) ([]byte, error) {
	buf := make([]byte, 9)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
