package websocket

import (
	"bytes"
	"testing"
)

func TestDefragmenterLifecycle(t *testing.T) {
	t.Parallel()
	d := &Defragmenter{}
	defer d.Reset()

	_, err := d.Write([]byte("hello "))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	_, err = d.Write([]byte("world"))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	payload := d.Payload()
	if !bytes.Equal(payload, []byte("hello world")) {
		t.Errorf("Expected 'hello world', got %s", string(payload))
	}
}

func TestDefragmenterBufferGrowth(t *testing.T) {
	t.Parallel()
	d := &Defragmenter{}
	defer d.Reset()

	largeData := bytes.Repeat([]byte("A"), 100000)
	_, err := d.Write(largeData)
	if err != nil {
		t.Fatalf("Unexpected error writing large payload: %v", err)
	}

	if !bytes.Equal(d.Payload(), largeData) {
		t.Error("Payload mismatch after buffer growth")
	}
}
