package transport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/Aditya-9-6/DevProxy/pkg/proxy/parser"
)

func TestChunkGuardVerify(t *testing.T) {
	tests := []struct {
		name string
		wire string
		want error
	}{
		{name: "empty", wire: "0\r\n\r\n"},
		{name: "body", wire: "4\r\ntest\r\n5\r\nvalue\r\n0\r\n\r\n"},
		{name: "trailers", wire: "4\r\ntest\r\n0\r\nX-Test: yes\r\n\r\n"},
		{name: "invalid-size", wire: "x\r\n", want: parser.ErrInvalidChunkSize},
		{name: "invalid-terminator", wire: "4\r\ntestXX", want: parser.ErrInvalidChunkTerminator},
		{name: "incomplete", wire: "4\r\nte", want: parser.ErrIncompleteChunk},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			guard := NewChunkGuard(UnlimitedChunkGuardLimits())
			err := guard.Verify(bytes.NewBufferString(test.wire))
			if test.want == nil && err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("Verify() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestChunkGuardSizeLimits(t *testing.T) {
	tests := []struct {
		name   string
		limits ChunkGuardLimits
		wire   string
	}{
		{name: "chunk", limits: ChunkGuardLimits{MaxChunkBytes: 3}, wire: "4\r\ntest\r\n0\r\n\r\n"},
		{name: "total", limits: ChunkGuardLimits{MaxTotalBytes: 3}, wire: "4\r\ntest\r\n0\r\n\r\n"},
		{name: "chunks", limits: ChunkGuardLimits{MaxChunks: 1}, wire: "1\r\na\r\n1\r\nb\r\n0\r\n\r\n"},
		{name: "encoded", limits: ChunkGuardLimits{MaxEncodedBytes: 5}, wire: "1\r\na\r\n0\r\n\r\n"},
		{name: "header", limits: ChunkGuardLimits{MaxHeaderBytes: 4}, wire: "4;x\r\ntest\r\n0\r\n\r\n"},
		{name: "trailer", limits: ChunkGuardLimits{MaxTrailerBytes: 4}, wire: "0\r\nX-Test: value\r\n\r\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			guard := NewChunkGuard(test.limits)
			err := guard.Verify(bytes.NewBufferString(test.wire))
			if !errors.Is(err, ErrChunkGuardLimit) {
				t.Fatalf("Verify() error = %v, want %v", err, ErrChunkGuardLimit)
			}
		})
	}
}

func TestChunkGuardConsumePreservesPipeline(t *testing.T) {
	body := []byte("4\r\ntest\r\n0\r\n\r\n")
	trailing := []byte("NEXT / HTTP/1.1\r\n")
	wire := append(append([]byte(nil), body...), trailing...)
	guard := NewChunkGuard(UnlimitedChunkGuardLimits())

	consumed, err := guard.Consume(context.Background(), wire)
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if consumed != len(body) {
		t.Fatalf("Consume() consumed = %d, want %d", consumed, len(body))
	}
	if !bytes.Equal(wire[consumed:], trailing) {
		t.Fatal("trailing pipeline bytes were not preserved")
	}
	stats := guard.Stats()
	if stats.ChunkCount != 1 || stats.DecodedBytes != 4 || !stats.Complete {
		t.Fatalf("Stats() = %+v", stats)
	}
}

func TestChunkGuardReaderForwardsTrailingPipeline(t *testing.T) {
	wire := []byte("4\r\ntest\r\n0\r\n\r\nNEXT / HTTP/1.1\r\n")
	guard := NewChunkGuard(UnlimitedChunkGuardLimits())
	reader := guard.WrapReader(bytes.NewReader(wire))
	buffer := make([]byte, len(wire))

	n, err := reader.Read(buffer)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if n != len(wire) {
		t.Fatalf("Read() count = %d, want %d", n, len(wire))
	}
	if !reader.BodyComplete() {
		t.Fatal("guarded body did not complete")
	}
	if !bytes.Equal(buffer, wire) {
		t.Fatal("reader did not preserve the raw pipeline bytes")
	}

	n, err = reader.Read(buffer[:1])
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("second Read() = %d, %v; want 0, EOF", n, err)
	}
}

type chunkGuardErrorReader struct {
	err error
}

func (r chunkGuardErrorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestChunkGuardNetworkAndContextErrors(t *testing.T) {
	networkErr := errors.New("network failed")
	guard := NewChunkGuard(UnlimitedChunkGuardLimits())
	if err := guard.Verify(chunkGuardErrorReader{err: networkErr}); !errors.Is(err, networkErr) {
		t.Fatalf("Verify() error = %v, want %v", err, networkErr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := guard.VerifyContext(ctx, bytes.NewReader([]byte("0\r\n\r\n"))); !errors.Is(err, context.Canceled) {
		t.Fatalf("VerifyContext() error = %v, want %v", err, context.Canceled)
	}
}

type recordingDeadlineReader struct {
	io.Reader
	mu        sync.Mutex
	deadlines []time.Time
}

func (r *recordingDeadlineReader) SetReadDeadline(deadline time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deadlines = append(r.deadlines, deadline)
	return nil
}

func (r *recordingDeadlineReader) deadlineCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.deadlines)
}

func TestChunkGuardReadDeadline(t *testing.T) {
	source := &recordingDeadlineReader{Reader: bytes.NewReader([]byte("0\r\n\r\n"))}
	guard := NewChunkGuard(ChunkGuardLimits{IdleTimeout: time.Second})
	if err := guard.Verify(source); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if source.deadlineCount() == 0 {
		t.Fatal("source did not receive a read deadline")
	}

	unsupported := NewChunkGuard(ChunkGuardLimits{IdleTimeout: time.Second})
	err := unsupported.Verify(bytes.NewReader([]byte("0\r\n\r\n")))
	if !errors.Is(err, ErrReadDeadlineUnsupported) {
		t.Fatalf("Verify() error = %v, want %v", err, ErrReadDeadlineUnsupported)
	}
}

func TestChunkGuardConcurrentStats(t *testing.T) {
	guard := NewChunkGuard(UnlimitedChunkGuardLimits())
	wire := []byte("1\r\na\r\n1\r\nb\r\n1\r\nc\r\n0\r\n\r\n")
	done := make(chan error, 1)
	go func() {
		_, err := guard.Consume(context.Background(), wire)
		done <- err
	}()

	for {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Consume() error = %v", err)
			}
			return
		default:
			_ = guard.Stats()
		}
	}
}

func TestChunkGuardCacheLineAlignment(t *testing.T) {
	if size := unsafe.Sizeof(cacheAlignedCounter{}); size != cacheLineSize {
		t.Fatalf("cacheAlignedCounter size = %d, want %d", size, cacheLineSize)
	}

	guard := NewChunkGuard(UnlimitedChunkGuardLimits())
	offsets := []uintptr{
		unsafe.Offsetof(guard.decodedBytes),
		unsafe.Offsetof(guard.encodedBytes),
		unsafe.Offsetof(guard.chunkCount),
		unsafe.Offsetof(guard.largestChunkBytes),
		unsafe.Offsetof(guard.rejectedChunks),
	}
	for _, offset := range offsets {
		if offset%cacheLineSize != 0 {
			t.Fatalf("counter offset = %d, want a multiple of %d", offset, cacheLineSize)
		}
	}
}

func TestChunkGuardConsumeDoesNotAllocate(t *testing.T) {
	guard := NewChunkGuard(UnlimitedChunkGuardLimits())
	wire := []byte("4\r\ntest\r\n0\r\n\r\n")
	allocs := testing.AllocsPerRun(100, func() {
		if err := guard.Reset(); err != nil {
			panic(err)
		}
		if _, err := guard.Consume(context.Background(), wire); err != nil {
			panic(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("Consume() allocations = %v, want 0", allocs)
	}
}
