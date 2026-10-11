package proxy

import (
	"bytes"
	"sync"
	"testing"
)

func TestH2Reassembler(t *testing.T) {
	pool := NewFramePool()
	reassembler := NewH2Reassembler(pool)

	data := []byte("hello world, this is a test of h2 reassembly")
	reader := bytes.NewReader(data)

	buf, err := reassembler.Reassemble(reader, len(data))
	if err != nil {
		t.Fatalf("Reassemble failed: %v", err)
	}

	if !bytes.Equal(buf, data) {
		t.Errorf("Expected %s, got %s", data, buf)
	}

	reassembler.Release(buf)
}

func TestH2Reassembler_Concurrency(t *testing.T) {
	t.Parallel()
	pool := NewFramePool()
	reassembler := NewH2Reassembler(pool)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := []byte("test data")
			buf, _ := reassembler.Reassemble(bytes.NewReader(data), len(data))
			reassembler.Release(buf)
		}()
	}
	wg.Wait()
}

func TestH2Reassembler_LargeAllocation(t *testing.T) {
	pool := NewFramePool()
	reassembler := NewH2Reassembler(pool)
	// Test allocation > 65536
	size := 70000
	data := make([]byte, size)
	buf, err := reassembler.Reassemble(bytes.NewReader(data), size)
	if err != nil {
		t.Fatalf("Large allocation failed: %v", err)
	}
	reassembler.Release(buf)
}
