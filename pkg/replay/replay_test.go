package replay

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestReplayRingBufferAndDuplication(t *testing.T) {
	ring := NewReplayRingBuffer(16, 64)

	payload := []byte("Hello, DevProxy Replay Engine! High-throughput atomic buffer test payload.")
	src := bytes.NewReader(payload)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dupReader, err := DuplicateAndReplay(ctx, src, ring)
	if err != nil {
		t.Fatalf("Failed to initialize duplication: %v", err)
	}

	// Read from duplicated pipe stream
	gotPipeline, err := io.ReadAll(dupReader)
	if err != nil && err != io.EOF {
		t.Fatalf("Error reading duplicated stream: %v", err)
	}

	if !bytes.Equal(payload, gotPipeline) {
		t.Errorf("Expected pipeline %q, got %q", payload, gotPipeline)
	}

	// Read from replay ring buffer via ReplayReader
	streamReader := NewReplayReader(ring)
	gotReplay, err := io.ReadAll(streamReader)
	if err != nil && err != io.EOF {
		t.Fatalf("Error reading replay buffer: %v", err)
	}

	if len(gotReplay) == 0 {
		t.Errorf("Expected non-empty replay buffer payload")
	}
}

func TestBufferPool(t *testing.T) {
	p := NewBufferPool(1024)
	buf := p.Get()
	if len(buf) != 1024 {
		t.Errorf("Expected buffer length 1024, got %d", len(buf))
	}
	p.Put(buf)
}
