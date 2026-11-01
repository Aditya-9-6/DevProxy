package replay

import (
	"io"
	"testing"
)

func TestReplayBuffer(t *testing.T) {
	buf := NewReplayBuffer(1024)
	data := []byte("hello devproxy")

	n, err := buf.Write(data)
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected written %d, got %d", len(data), n)
	}

	readBytes := buf.Bytes()
	if string(readBytes) != string(data) {
		t.Errorf("expected %s, got %s", data, readBytes)
	}

	rdr := buf.Reader()
	readAll, err := io.ReadAll(rdr)
	if err != nil {
		t.Fatalf("failed to read all: %v", err)
	}
	if string(readAll) != string(data) {
		t.Errorf("expected reader data %s, got %s", data, readAll)
	}

	buf.Reset()
	if len(buf.Bytes()) != 0 {
		t.Errorf("expected empty buffer after reset, got len %d", len(buf.Bytes()))
	}
}

func TestBufferPool(t *testing.T) {
	buf := AcquireBuffer()
	if buf == nil {
		t.Fatal("expected non-nil buffer from pool")
	}
	_, err := buf.Write([]byte("pooled"))
	if err != nil {
		t.Fatalf("failed to write to pooled buffer: %v", err)
	}
	ReleaseBuffer(buf)
}
