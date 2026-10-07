package proxy

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
)

var grpcBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

// GRPCFrame represents a parsed gRPC message frame.
type GRPCFrame struct {
	Compressed bool
	Length     uint32
	Data       []byte
}

// ParseGRPCStream reads gRPC frames from an io.Reader.
func ParseGRPCStream(r io.Reader) ([]GRPCFrame, error) {
	var frames []GRPCFrame
	buf := grpcBufferPool.Get().([]byte)
	defer grpcBufferPool.Put(buf)

	for {
		header := make([]byte, 5)
		_, err := io.ReadFull(r, header)
		if err == io.EOF {
			break
		}
		if err != nil {
			return frames, err
		}

		compressed := header[0] != 0
		length := binary.BigEndian.Uint32(header[1:])

		data := make([]byte, length)
		_, err = io.ReadFull(r, data)
		if err != nil {
			return frames, err
		}

		frames = append(frames, GRPCFrame{
			Compressed: compressed,
			Length:     length,
			Data:       data,
		})
	}
	return frames, nil
}
