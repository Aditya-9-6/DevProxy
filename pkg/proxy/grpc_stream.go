package proxy

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
)

const (
	// grpcHeaderLength is the 5-byte gRPC length-prefixed header size.
	grpcHeaderLength = 5
)

var (
	// ErrOversizedMessage is returned when a gRPC message exceeds MaxGRPCMessageSize.
	ErrOversizedMessage = errors.New("grpc message exceeds maximum allowed size")

	// grpcBufferPool pools byte slices for gRPC message payloads to enforce zero-allocation streaming.
	grpcBufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 1024)
		},
	}
)

// Deframer parses gRPC length-prefixed messages from a stream.
type Deframer struct {
	reader io.Reader
}

// NewDeframer creates a new gRPC Deframer.
func NewDeframer(r io.Reader) *Deframer {
	return &Deframer{reader: r}
}

// Next reads the next gRPC message from the stream without context cancellation.
func (d *Deframer) Next() (*GRPCMessage, error) {
	return d.NextContext(context.Background())
}

// NextContext reads the next gRPC message from the stream with context cancellation support.
func (d *Deframer) NextContext(ctx context.Context) (*GRPCMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	header := make([]byte, grpcHeaderLength)
	if err := readFullContext(ctx, d.reader, header); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read grpc header: %w", err)
	}

	compressed := header[0] == 1
	length := binary.BigEndian.Uint32(header[1:])

	if length > MaxGRPCMessageSize {
		return nil, fmt.Errorf("%w: size %d exceeds limit %d", ErrOversizedMessage, length, MaxGRPCMessageSize)
	}

	dataBuf := allocateBuffer(int(length))
	dataBytes := dataBuf[:length]

	if err := readFullContext(ctx, d.reader, dataBytes); err != nil {
		releaseBuffer(dataBuf)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read grpc payload: %w", err)
	}

	return &GRPCMessage{
		Compressed: compressed,
		Data:       dataBytes,
	}, nil
}

// Release returns a GRPCMessage's data buffer back to the pool if applicable.
func Release(msg *GRPCMessage) {
	if msg != nil && msg.Data != nil {
		releaseBuffer(msg.Data)
	}
}

func allocateBuffer(length int) []byte {
	if length == 0 {
		return nil
	}
	buf := grpcBufferPool.Get().([]byte)
	if cap(buf) < length {
		return make([]byte, length)
	}
	return buf[:length]
}

func releaseBuffer(buf []byte) {
	if buf == nil {
		return
	}
	clear(buf)
	grpcBufferPool.Put(buf[:0])
}

func readFullContext(ctx context.Context, r io.Reader, buf []byte) error {
	totalRead := 0
	for totalRead < len(buf) {
		if err := ctx.Err(); err != nil {
			return err
		}

		n, err := r.Read(buf[totalRead:])
		totalRead += n

		if err != nil {
			if totalRead == len(buf) {
				return nil
			}
			return err
		}
		if n == 0 && totalRead < len(buf) {
			return io.ErrUnexpectedEOF
		}
	}
	return nil
}
