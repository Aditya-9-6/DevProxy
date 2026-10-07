package proxy

import (
	"encoding/binary"
	"errors"
	"io"
)

// GRPCFrame represents a single gRPC message frame.
type GRPCFrame struct {
	Compressed bool
	Length     uint32
	Data       []byte
}

// ParseGRPCFrame reads a single gRPC frame from the reader.
func ParseGRPCFrame(r io.Reader) (*GRPCFrame, error) {
	header := make([]byte, 5)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return nil, err
	}

	compressed := header[0] == 1
	length := binary.BigEndian.Uint32(header[1:5])

	data := make([]byte, length)
	if length > 0 {
		_, err = io.ReadFull(r, data)
		if err != nil {
			return nil, err
		}
	}

	return &GRPCFrame{
		Compressed: compressed,
		Length:     length,
		Data:       data,
	}, nil
}
