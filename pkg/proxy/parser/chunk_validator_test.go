package parser

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

func TestChunkValidatorValidStreams(t *testing.T) {
	tests := []struct {
		name      string
		wire      string
		chunks    uint64
		total     uint64
		lastChunk uint64
	}{
		{name: "empty", wire: "0\r\n\r\n", chunks: 0, total: 0},
		{name: "single", wire: "4\r\nWiki\r\n0\r\n\r\n", chunks: 1, total: 4, lastChunk: 4},
		{name: "multiple", wire: "4\r\nWiki\r\n5\r\npedia\r\n0\r\n\r\n", chunks: 2, total: 9, lastChunk: 5},
		{name: "extensions", wire: "4;foo=bar\r\nWiki\r\n0;done=yes\r\n\r\n", chunks: 1, total: 4, lastChunk: 4},
		{name: "quoted-extension", wire: "4;foo=\"bar baz\"\r\nWiki\r\n0\r\n\r\n", chunks: 1, total: 4, lastChunk: 4},
		{name: "trailers", wire: "4\r\nWiki\r\n0\r\nX-Test: yes\r\n\r\n", chunks: 1, total: 4, lastChunk: 4},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := NewChunkValidator(UnlimitedChunkLimits())
			consumed, err := validator.Consume([]byte(test.wire))
			if err != nil {
				t.Fatalf("Consume() error = %v", err)
			}
			if consumed != len(test.wire) {
				t.Fatalf("Consume() consumed = %d, want %d", consumed, len(test.wire))
			}
			if !validator.Complete() {
				t.Fatal("validator did not reach the message boundary")
			}
			stats := validator.Stats()
			if stats.ChunkCount != test.chunks || stats.TotalBytes != test.total || stats.LastChunkBytes != test.lastChunk {
				t.Fatalf("Stats() = %+v, want chunks=%d total=%d last=%d", stats, test.chunks, test.total, test.lastChunk)
			}
		})
	}
}

func TestChunkValidatorMalformedStreams(t *testing.T) {
	tests := []struct {
		name string
		wire string
		want error
	}{
		{name: "missing-size", wire: "\r\n", want: ErrInvalidChunkSize},
		{name: "invalid-size", wire: "z\r\n", want: ErrInvalidChunkSize},
		{name: "size-overflow", wire: "fffffffffffffffff\r\n", want: ErrChunkSizeOverflow},
		{name: "bare-lf", wire: "4\nWiki\r\n", want: ErrInvalidChunkTerminator},
		{name: "bad-data-terminator", wire: "4\r\nWikiXX", want: ErrInvalidChunkTerminator},
		{name: "bad-extension", wire: "4;bad name=x\r\nWiki\r\n", want: ErrInvalidChunkExtension},
		{name: "bad-trailer-name", wire: "0\r\nBad Header: value\r\n\r\n", want: ErrInvalidTrailer},
		{name: "missing-final-crlf", wire: "0\r\n", want: ErrIncompleteChunk},
		{name: "incomplete-data", wire: "4\r\nWi", want: ErrIncompleteChunk},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := NewChunkValidator(UnlimitedChunkLimits())
			err := validator.Validate(bytes.NewBufferString(test.wire))
			if !errors.Is(err, test.want) {
				t.Fatalf("Validate() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestChunkValidatorLimits(t *testing.T) {
	tests := []struct {
		name   string
		limits ChunkLimits
		wire   string
		want   error
	}{
		{name: "chunk", limits: ChunkLimits{MaxChunkBytes: 3}, wire: "4\r\ntest\r\n0\r\n\r\n", want: ErrChunkTooLarge},
		{name: "total", limits: ChunkLimits{MaxTotalBytes: 3}, wire: "4\r\ntest\r\n0\r\n\r\n", want: ErrBodyTooLarge},
		{name: "chunks", limits: ChunkLimits{MaxChunks: 1}, wire: "1\r\na\r\n1\r\nb\r\n0\r\n\r\n", want: ErrTooManyChunks},
		{name: "header", limits: ChunkLimits{MaxHeaderBytes: 4}, wire: "4;x\r\ntest\r\n0\r\n\r\n", want: ErrChunkHeaderTooLarge},
		{name: "trailer", limits: ChunkLimits{MaxTrailerBytes: 4}, wire: "0\r\nX-Test: value\r\n\r\n", want: ErrTrailerTooLarge},
		{name: "encoded", limits: ChunkLimits{MaxEncodedBytes: 5}, wire: "1\r\na\r\n0\r\n\r\n", want: ErrEncodedBodyTooLarge},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := NewChunkValidator(test.limits)
			err := validator.Validate(bytes.NewBufferString(test.wire))
			if !errors.Is(err, test.want) {
				t.Fatalf("Validate() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestChunkValidatorObserver(t *testing.T) {
	observerErr := errors.New("observer rejected chunk")
	validator := NewChunkValidatorWithObserver(UnlimitedChunkLimits(), ChunkObserverFunc(func(event ChunkEvent) error {
		if event.Index != 1 || event.Size != 4 || event.TotalBytes != 4 {
			t.Fatalf("unexpected event: %+v", event)
		}
		return observerErr
	}))

	_, err := validator.Consume([]byte("4\r\n"))
	if !errors.Is(err, observerErr) {
		t.Fatalf("Consume() error = %v, want %v", err, observerErr)
	}
}

func TestChunkValidatorPreservesPipelineBoundary(t *testing.T) {
	body := []byte("4\r\ntest\r\n0\r\n\r\n")
	trailing := []byte("GET /next HTTP/1.1\r\n")
	wire := append(append([]byte(nil), body...), trailing...)

	validator := NewChunkValidator(UnlimitedChunkLimits())
	consumed, err := validator.Consume(wire)
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if consumed != len(body) {
		t.Fatalf("Consume() consumed = %d, want %d", consumed, len(body))
	}
	if !validator.Complete() {
		t.Fatal("validator did not complete at the pipeline boundary")
	}
	if !bytes.Equal(wire[consumed:], trailing) {
		t.Fatal("trailing pipeline bytes were not preserved")
	}
}

type chunkValidatorErrorReader struct {
	err error
}

func (r chunkValidatorErrorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestChunkValidatorNetworkAndContextErrors(t *testing.T) {
	networkErr := errors.New("network failed")
	validator := NewChunkValidator(UnlimitedChunkLimits())
	if err := validator.Validate(chunkValidatorErrorReader{err: networkErr}); !errors.Is(err, networkErr) {
		t.Fatalf("Validate() error = %v, want %v", err, networkErr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := validator.ValidateContext(ctx, bytes.NewReader([]byte("0\r\n\r\n"))); !errors.Is(err, context.Canceled) {
		t.Fatalf("ValidateContext() error = %v, want %v", err, context.Canceled)
	}
}

func TestChunkValidatorConcurrentStats(t *testing.T) {
	validator := NewChunkValidator(UnlimitedChunkLimits())
	done := make(chan error, 1)
	go func() {
		_, err := validator.Consume([]byte("4\r\ntest\r\n5\r\nvalue\r\n0\r\n\r\n"))
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
			_ = validator.Stats()
		}
	}
}

func TestChunkValidatorConsumeDoesNotAllocate(t *testing.T) {
	validator := NewChunkValidator(UnlimitedChunkLimits())
	wire := []byte("4\r\ntest\r\n0\r\n\r\n")
	allocs := testing.AllocsPerRun(100, func() {
		if err := validator.Reset(); err != nil {
			panic(err)
		}
		if _, err := validator.Consume(wire); err != nil {
			panic(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("Consume() allocations = %v, want 0", allocs)
	}
}

var _ io.Reader = chunkValidatorErrorReader{}
