package wscompress

import (
	"bytes"
	"compress/flate"
	"context"
	"io"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

// StreamEngine handles RFC 7692 message compression and decompression streaming.
type StreamEngine struct {
	limits    Limits
	validator *Validator
}

// NewStreamEngine creates a new streaming engine.
func NewStreamEngine(l Limits) *StreamEngine {
	return &StreamEngine{
		limits:    l,
		validator: NewValidator(l),
	}
}

// Compress compresses raw payload data appending RFC 7692 tail 00 00 ff ff if needed.
func (e *StreamEngine) Compress(ctx context.Context, src io.Reader) ([]byte, error) {
	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)

	var out bytes.Buffer
	w, err := flate.NewWriter(&out, flate.BestSpeed)
	if err != nil {
		return nil, err
	}

	var totalRead int
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		n, rErr := src.Read(*bufPtr)
		if n > 0 {
			totalRead += n
			if _, wErr := w.Write((*bufPtr)[:n]); wErr != nil {
				_ = w.Close()
				return nil, wErr
			}
			if err := e.validator.Track(n, out.Len()); err != nil {
				_ = w.Close()
				return nil, err
			}
		}
		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			_ = w.Close()
			return nil, rErr
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	compressed := out.Bytes()
	// Strip 4-byte DEFLATE tail 00 00 ff ff as required by RFC 7692 section 7.2.1
	if bytes.HasSuffix(compressed, []byte{0x00, 0x00, 0xff, 0xff}) {
		compressed = compressed[:len(compressed)-4]
	}

	return compressed, nil
}

// Decompress decompresses an RFC 7692 compressed WebSocket message payload.
func (e *StreamEngine) Decompress(ctx context.Context, payload []byte) ([]byte, error) {
	// Append 00 00 ff ff tail back for standard flate reader
	fullData := make([]byte, len(payload)+4)
	copy(fullData, payload)
	copy(fullData[len(payload):], []byte{0x00, 0x00, 0xff, 0xff})

	r := flate.NewReader(bytes.NewReader(fullData))
	defer r.Close()

	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)

	var out bytes.Buffer
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		n, err := r.Read(*bufPtr)
			if n > 0 {
			if _, wErr := out.Write((*bufPtr)[:n]); wErr != nil {
				return nil, wErr
			}
			if err := e.validator.Track(len(payload), out.Len()); err != nil {
				return nil, err
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	return out.Bytes(), nil
}
