package replay

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
)

func TestConcurrentReadWrite(t *testing.T) {
	ring := NewReplayRingBuffer(10, 1024)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			ring.WriteChunk([]byte("data"))
		}
	}()
	go func() {
		defer wg.Done()
		r := NewReplayReader(ring)
		buf := make([]byte, 1024)
		for i := 0; i < 100; i++ {
			_, _ = r.Read(buf)
		}
	}()
	wg.Wait()
}

func TestReplayRingBufferAndDuplication(t *testing.T) {
	ring := NewReplayRingBuffer(16, 64)
	payload := []byte("Hello, DevProxy Replay Engine!")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dupReader, _ := DuplicateAndReplay(ctx, bytes.NewReader(payload), ring)
	got, _ := io.ReadAll(dupReader)
	if !bytes.Equal(payload, got) {
		t.Errorf("mismatch")
	}
}
