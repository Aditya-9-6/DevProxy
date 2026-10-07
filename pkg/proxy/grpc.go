package proxy

import (
	"encoding/binary"
	"errors"
	"io"
)

// GRPCFrame represents a single gRPC message chunk.
type GRPCFrame struct {
	Compressed bool
	Length     uint32
	Data       []byte
}

// ParseGRPCFrame reads a single gRPC frame from the stream.
// Format: 1 byte (compressed flag) + 4 bytes (big-endian length).
func ParseGRPCFrame(r io.Reader) (*GRPCFrame, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	frame := &GRPCFrame{
		Compressed: header[0] == 1,
		Length:     binary.BigEndian.Uint32(header[1:]),
	}

	if frame.Length > 0 {
		frame.Data = make([]byte, frame.Length)
		if _, err := io.ReadFull(r, frame.Data); err != nil {
			return nil, err
		}
	}
	return frame, nil
}
