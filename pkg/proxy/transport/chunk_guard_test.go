package transport

import (
	"bytes"
	"context"
	"testing"
)

func TestChunkGuard_NormalOperation(t *testing.T) {
	data := []byte("hello devproxy chunked stream")
	guard := NewChunkGuard(1024, 256)
	ctx := context.Background()

	reader := guard.WrapReader(ctx, bytes.NewReader(data))
	buf := make([]byte, 128)

	n, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}

	if string(buf[:n]) != string(data) {
		t.Errorf("expected %q, got %q", string(data), string(buf[:n]))
	}

	_, chunks, _ := guard.Stats()
	if chunks == 0 {
		t.Errorf("expected chunks processed > 0, got %d", chunks)
	}
}

func TestChunkGuard_MaxTotalExceeded(t *testing.T) {
	data := []byte("this stream exceeds maximum total size quota")
	guard := NewChunkGuard(10, 100)
	ctx := context.Background()

	reader := guard.WrapReader(ctx, bytes.NewReader(data))
	buf := make([]byte, 64)

	_, err := reader.Read(buf)
	if err != ErrMaxTotalExceeded {
		t.Errorf(
			"expected ErrMaxTotalExceeded, got %v",
			err,
		)
	}
}

func TestChunkGuard_ContextCancellation(t *testing.T) {
	guard := NewChunkGuard(1000, 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	reader := guard.WrapReader(ctx, bytes.NewReader([]byte("test")))
	buf := make([]byte, 10)

	_, err := reader.Read(buf)
	if err == nil {
		t.Errorf("expected context cancellation error, got nil")
	}
}
