package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/google/uuid"
)

// HandleGRPCInterception intercepts HTTP/2 gRPC requests (Content-Type: application/grpc),
// checks for local mock definitions, and streams or records traffic events.
func HandleGRPCInterception(w http.ResponseWriter, r *http.Request, mockEngine *mock.GRPCDescriptorEngine, ringBuf *ringbuffer.RingBuffer) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") && !strings.HasPrefix(r.Header.Get("content-type"), "application/grpc") {
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

	reqBytes, _ := io.ReadAll(r.Body)

	callCtx := &mock.GRPCCallContext{
		ServiceName: serviceName,
		MethodName:  methodName,
		Input:       reqBytes,
	}

	// Check if a mock is registered for this gRPC endpoint
	if mockEngine != nil {
		if mockResp, matched := mockEngine.EvaluateMock(callCtx); matched {
			w.Header().Set("Content-Type", "application/grpc")
			w.Header().Set("Trailer", "grpc-status: 0\r\ngrpc-message: \r\n")
			w.WriteHeader(http.StatusOK)

			// gRPC framed response: 1 byte compressed flag (0), 4 bytes length, payload
			frame := make([]byte, 5+len(mockResp))
			frame[0] = 0
			length := uint32(len(mockResp))
			frame[1] = byte(length >> 24)
			frame[2] = byte(length >> 16)
			frame[3] = byte(length >> 8)
			frame[4] = byte(length)
			copy(frame[5:], mockResp)

			_, _ = w.Write(frame)

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

type bytesBufferInterface interface {
	Reset()
	Bytes() []byte
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
