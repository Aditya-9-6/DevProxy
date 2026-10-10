package proxy

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/google/uuid"
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

// HandleGRPCInterception intercepts HTTP/2 gRPC requests (Content-Type: application/grpc),
// checks for local mock definitions, and streams or records traffic events.
func HandleGRPCInterception(w http.ResponseWriter, r *http.Request, mockEngine *mock.GRPCDescriptorEngine, ringBuf *ringbuffer.RingBuffer) bool {
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/grpc") {
		return false
	}

	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	var serviceName, methodName string
	if len(parts) >= 2 {
		serviceName = parts[0]
		methodName = parts[1]
	}

	// Read request body using zero-allocation buffer from pool
	buf := mock.GRPCPayloadPool.Get().(*bytes.Buffer)
	defer mock.GRPCPayloadPool.Put(buf)
	buf.Reset()
	_, _ = io.Copy(buf, r.Body)
	reqBytes := buf.Bytes()

	callCtx := &mock.GRPCCallContext{
		ServiceName: serviceName,
		MethodName:  methodName,
		Input:       reqBytes,
	}

	// Check if a mock is registered for this gRPC endpoint
	if mockEngine != nil {
		if mockResp, matched := mockEngine.EvaluateMock(callCtx); matched {
			w.Header().Set("Content-Type", "application/grpc")
			w.Header().Set("Trailer", "grpc-status, grpc-message")
			w.WriteHeader(http.StatusOK)

			msg := &GRPCMessage{
				Compressed: false,
				Data:       mockResp,
			}
			_ = WriteGRPCMessage(w, msg)

			// Set trailers for gRPC status
			w.Header().Set("grpc-status", "0")
			w.Header().Set("grpc-message", "")

			if ringBuf != nil {
				event := &ringbuffer.TrafficEvent{
					ID:         uuid.New().String(),
					Method:     "POST",
					URL:        r.URL.String(),
					Host:       r.Host,
					Path:       r.URL.Path,
					StatusCode: 200,
					ReqBody:    reqBytes,
					RespBody:   mockResp,
				}
				ringBuf.Push(event)
			}
			return true
		}
	}

	return false
}

// GRPCStreamInterceptor handles bidirectional streaming gRPC context lifecycle.
func GRPCStreamInterceptor(ctx context.Context, serviceName, methodName string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// WriteGRPCMessage writes a framed gRPC message to w.
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
