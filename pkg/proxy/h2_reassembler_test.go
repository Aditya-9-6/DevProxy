package proxy

import (
	"bytes"
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
