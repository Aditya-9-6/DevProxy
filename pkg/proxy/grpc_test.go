package proxy

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestGRPCFramingRoundTrip(t *testing.T) {
	originalMsg := &GRPCMessage{
		Compressed: false,
		Data:       []byte("hello grpc stream"),
	}

	var buf bytes.Buffer
	if err := WriteGRPCMessage(&buf, originalMsg); err != nil {
		t.Fatalf("Failed to write gRPC message: %v", err)
	}

	readMsg, err := ReadGRPCMessage(&buf)
	if err != nil {
		t.Fatalf("Failed to read gRPC message: %v", err)
	}

	if readMsg.Compressed != originalMsg.Compressed {
		t.Errorf("Compressed flag mismatch: expected %v, got %v", originalMsg.Compressed, readMsg.Compressed)
	}

	if string(readMsg.Data) != string(originalMsg.Data) {
		t.Errorf("Payload mismatch: expected %s, got %s", string(originalMsg.Data), string(readMsg.Data))
	}
}

func TestGRPCFramingCompressedAndEmpty(t *testing.T) {
	msg := &GRPCMessage{
		Compressed: true,
		Data:       []byte{},
	}

	var buf bytes.Buffer
	if err := WriteGRPCMessage(&buf, msg); err != nil {
		t.Fatalf("Failed to write compressed empty message: %v", err)
	}

	readMsg, err := ReadGRPCMessage(&buf)
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if !readMsg.Compressed {
		t.Errorf("Expected compressed=true, got %v", readMsg.Compressed)
	}
	if len(readMsg.Data) != 0 {
		t.Errorf("Expected empty data, got %v", readMsg.Data)
	}
}

func TestGRPCFramingErrors(t *testing.T) {
	if err := WriteGRPCMessage(&bytes.Buffer{}, nil); err == nil {
		t.Error("Expected error writing nil message, got nil")
	}

	// Truncated header
	shortBuf := bytes.NewReader([]byte{0, 0, 0})
	if _, err := ReadGRPCMessage(shortBuf); err == nil {
		t.Errorf("Expected error reading truncated header, got nil")
	}

	// Truncated payload: claims length 10, but only 2 bytes present
	corruptBuf := bytes.NewReader([]byte{0, 0, 0, 0, 10, 1, 2})
	if _, err := ReadGRPCMessage(corruptBuf); err == nil {
		t.Error("Expected error reading truncated payload, got nil")
	}

	// Oversized message frame (> 64MB)
	oversizedBuf := bytes.NewReader([]byte{0, 0x05, 0x00, 0x00, 0x01}) // 83MB > 64MB
	if _, err := ReadGRPCMessage(oversizedBuf); !errors.Is(err, ErrGRPCMessageTooLarge) {
		t.Errorf("Expected ErrGRPCMessageTooLarge, got %v", err)
	}
}

func TestGRPCStreamInterceptor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	if err := GRPCStreamInterceptor(ctx, "TestService", "TestMethod"); err != nil {
		t.Fatalf("expected nil error for active context, got %v", err)
	}

	cancel()
	if err := GRPCStreamInterceptor(ctx, "TestService", "TestMethod"); err == nil {
		t.Fatalf("expected context cancellation error, got nil")
	}
}

func TestHandleGRPCInterception(t *testing.T) {
	mockEngine := mock.NewGRPCDescriptorEngine()
	mockEngine.RegisterMock("/greet.Greeter/SayHello", []byte("mocked reply"))
	ringBuf := ringbuffer.NewRingBuffer(100)

	// Case 1: Non-gRPC Content-Type returns false
	req := httptest.NewRequest(http.MethodPost, "/greet.Greeter/SayHello", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	if HandleGRPCInterception(w, req, mockEngine, ringBuf) {
		t.Errorf("expected false for application/json")
	}

	// Case 2: Matching mock endpoint returns true with gRPC response
	var framedReq bytes.Buffer
	_ = WriteGRPCMessage(&framedReq, &GRPCMessage{Data: []byte("ping")})

	req = httptest.NewRequest(http.MethodPost, "/greet.Greeter/SayHello", &framedReq)
	req.Header.Set("Content-Type", "application/grpc")
	w = httptest.NewRecorder()

	if !HandleGRPCInterception(w, req, mockEngine, ringBuf) {
		t.Fatalf("expected true for matched gRPC mock")
	}

	if w.Header().Get("Content-Type") != "application/grpc" {
		t.Errorf("expected Content-Type application/grpc, got %s", w.Header().Get("Content-Type"))
	}

	// Read and verify mocked response message
	respMsg, err := ReadGRPCMessage(w.Body)
	if err != nil {
		t.Fatalf("failed to read mocked response: %v", err)
	}
	if string(respMsg.Data) != "mocked reply" {
		t.Errorf("response data mismatch: expected 'mocked reply', got '%s'", string(respMsg.Data))
	}
}

func FuzzGRPCFraming(f *testing.F) {
	f.Add(true, []byte("sample payload"))
	f.Add(false, []byte("another payload"))

	f.Fuzz(func(t *testing.T, compressed bool, data []byte) {
		msg := &GRPCMessage{
			Compressed: compressed,
			Data:       data,
		}
		var buf bytes.Buffer
		if err := WriteGRPCMessage(&buf, msg); err == nil {
			_, _ = ReadGRPCMessage(&buf)
		}
	})
}

func BenchmarkGRPCFraming_WriteAndRead(b *testing.B) {
	msg := &GRPCMessage{
		Compressed: false,
		Data:       []byte("sample benchmark grpc message payload"),
	}
	var buf bytes.Buffer
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = WriteGRPCMessage(&buf, msg)
		_, _ = ReadGRPCMessage(&buf)
	}
}
