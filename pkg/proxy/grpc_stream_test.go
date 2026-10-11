package proxy

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"time"
)

func TestDeframer_ValidFrame(t *testing.T) {
	payload := []byte("hello grpc")
	frame := make([]byte, 5+len(payload))
	frame[0] = 0 // not compressed
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	copy(frame[5:], payload)

	deframer := NewDeframer(bytes.NewReader(frame))
	msg, err := deframer.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer Release(msg)

	if msg.Compressed {
		t.Errorf("expected uncompressed, got compressed")
	}
	if !bytes.Equal(msg.Data, payload) {
		t.Errorf("expected payload %q, got %q", payload, msg.Data)
	}
}

func TestDeframer_CompressedFrame(t *testing.T) {
	payload := []byte("compressed payload")
	frame := make([]byte, 5+len(payload))
	frame[0] = 1 // compressed
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	copy(frame[5:], payload)

	deframer := NewDeframer(bytes.NewReader(frame))
	msg, err := deframer.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer Release(msg)

	if !msg.Compressed {
		t.Errorf("expected compressed, got uncompressed")
	}
	if !bytes.Equal(msg.Data, payload) {
		t.Errorf("expected payload %q, got %q", payload, msg.Data)
	}
}

func TestDeframer_OversizedHeader(t *testing.T) {
	frame := make([]byte, 5)
	frame[0] = 0
	binary.BigEndian.PutUint32(frame[1:5], MaxGRPCMessageSize+1)

	deframer := NewDeframer(bytes.NewReader(frame))
	_, err := deframer.Next()
	if err == nil {
		t.Fatalf("expected error for oversized message, got nil")
	}
	if !errors.Is(err, ErrOversizedMessage) {
		t.Errorf("expected ErrOversizedMessage, got %v", err)
	}
}

func TestDeframer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deframer := NewDeframer(bytes.NewReader(nil))
	_, err := deframer.NextContext(ctx)
	if err == nil {
		t.Fatalf("expected cancellation error, got nil")
	}
}

func TestDeframer_FragmentedRead(t *testing.T) {
	payload := []byte("fragmented data")
	frame := make([]byte, 5+len(payload))
	frame[0] = 0
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	copy(frame[5:], payload)

	slowReader := &slowChunkReader{data: frame, chunkSize: 2}
	deframer := NewDeframer(slowReader)

	msg, err := deframer.Next()
	if err != nil {
		t.Fatalf("unexpected error on fragmented read: %v", err)
	}
	defer Release(msg)

	if !bytes.Equal(msg.Data, payload) {
		t.Errorf("expected payload %q, got %q", payload, msg.Data)
	}
}

type slowChunkReader struct {
	data      []byte
	offset    int
	chunkSize int
}

func (s *slowChunkReader) Read(p []byte) (int, error) {
	if s.offset >= len(s.data) {
		return 0, io.EOF
	}

	end := s.offset + s.chunkSize
	if end > len(s.data) {
		end = len(s.data)
	}

	n := copy(p, s.data[s.offset:end])
	s.offset += n
	time.Sleep(time.Millisecond)
	return n, nil
}
