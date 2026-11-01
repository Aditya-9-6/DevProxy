package transport

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestChunkGuard_NormalOperation(t *testing.T) {
	g := NewChunkGuard(1024, 5000)
	data := "hello chunked transfer encoding"
	r := g.WrapReader(context.Background(), bytes.NewReader([]byte(data)))

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(out) != data {
		t.Errorf("expected %q, got %q", data, string(out))
	}

	if g.TotalRead() != int64(len(data)) {
		t.Errorf("expected total read %d, got %d", len(data), g.TotalRead())
	}
}

func TestChunkGuard_MaxChunkSizeExceeded(t *testing.T) {
	g := NewChunkGuard(5, 100)
	data := "toolongchunk"
	r := g.WrapReader(context.Background(), bytes.NewReader([]byte(data)))

	_, err := io.ReadAll(r)
	if err == nil {
		t.Fatalf("expected chunk size exceeded error, got nil")
	}
}

func TestChunkGuard_MaxTotalSizeExceeded(t *testing.T) {
	g := NewChunkGuard(100, 10)
	data := "this is way too large overall"
	r := g.WrapReader(context.Background(), bytes.NewReader([]byte(data)))

	_, err := io.ReadAll(r)
	if err == nil {
		t.Fatalf("expected total size exceeded error, got nil")
	}
}

func TestChunkGuard_ContextCancellation(t *testing.T) {
	g := NewChunkGuard(1000, 5000)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := g.WrapReader(ctx, bytes.NewReader([]byte("data")))
	_, err := io.ReadAll(r)
	if err == nil {
		t.Fatalf("expected error on cancelled context, got nil")
	}
}
