package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
)

// ErrGRPCMessageTooLarge indicates that a message frame exceeds maximum size.
var ErrGRPCMessageTooLarge = errors.New("grpc message exceeds maximum size")

// MaxGRPCMessageSize defines the maximum framed payload size (64MB) to prevent OOM.
const MaxGRPCMessageSize = 64 * 1024 * 1024

// GRPCMessage represents a framed gRPC message.
type GRPCMessage struct {
	Compressed bool
	Data       []byte
}

var grpcHeaderPool = sync.Pool{
	New: func() any {
		b := make([]byte, 5)
		return &b
	},
}

// WriteGRPCMessage writes a framed gRPC message to w.
// Framing specification:
// - 1 byte compression flag (0 = uncompressed, 1 = compressed)
// - 4 bytes big-endian unsigned integer indicating payload length
// - N bytes payload data
func WriteGRPCMessage(w io.Writer, msg *GRPCMessage) error {
	if msg == nil {
		return errors.New("grpc: nil message")
	}

	hPtr := grpcHeaderPool.Get().(*[]byte)
	header := *hPtr
	defer grpcHeaderPool.Put(hPtr)

	if msg.Compressed {
		header[0] = 1
	} else {
		header[0] = 0
	}
	binary.BigEndian.PutUint32(header[1:5], uint32(len(msg.Data)))

	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write grpc header: %w", err)
	}
	if len(msg.Data) > 0 {
		if _, err := w.Write(msg.Data); err != nil {
			return fmt.Errorf("failed to write grpc payload: %w", err)
		}
	}
	return nil
}

// ReadGRPCMessage parses and decodes a single framed gRPC message from r.
func ReadGRPCMessage(r io.Reader) (*GRPCMessage, error) {
	hPtr := grpcHeaderPool.Get().(*[]byte)
	header := *hPtr
	defer grpcHeaderPool.Put(hPtr)

	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	compressed := header[0] == 1
	length := binary.BigEndian.Uint32(header[1:5])

	if length > MaxGRPCMessageSize {
		return nil, ErrGRPCMessageTooLarge
	}

	data := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, fmt.Errorf("failed to read grpc payload: %w", err)
		}
	}

	return &GRPCMessage{
		Compressed: compressed,
		Data:       data,
	}, nil
}
