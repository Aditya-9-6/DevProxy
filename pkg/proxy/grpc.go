package proxy

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
)

const MaxGRPCFrameSize = 4 * 1024 * 1024 // 4MB limit

var grpcBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, MaxGRPCFrameSize)
	},
}

// GRPCFrame represents a parsed gRPC message frame.
type GRPCFrame struct {
	Compressed bool
	Length     uint32
	Data       []byte
}

// ParseGRPCStream reads gRPC frames from an io.Reader using a callback to support streaming.
func ParseGRPCStream(r io.Reader, callback func(GRPCFrame) error) error {
	buf := grpcBufferPool.Get().([]byte)
	defer grpcBufferPool.Put(buf)

	for {
		header := make([]byte, 5)
		_, err := io.ReadFull(r, header)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		compressed := header[0] != 0
		length := binary.BigEndian.Uint32(header[1:])

		if length > MaxGRPCFrameSize {
			return errors.New("gRPC frame exceeds maximum size")
		}

		// Use pooled buffer for reading data
		_, err = io.ReadFull(r, buf[:length])
		if err != nil {
			return err
		}

		frameData := make([]byte, length)
		copy(frameData, buf[:length])

		if err := callback(GRPCFrame{Compressed: compressed, Length: length, Data: frameData}); err != nil {
			return err
		}
	}
}
