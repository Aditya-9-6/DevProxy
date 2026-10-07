package proxy

import (
	"bytes"
	"testing"
)

func TestParseGRPCFrame(t *testing.T) {
	data := []byte("hello")
	buf := new(bytes.Buffer)
	buf.Write([]byte{0, 0, 0, 0, 5})
	buf.Write(data)

	frame, err := ParseGRPCFrame(buf)
	if err != nil {
		t.Fatalf("failed to parse frame: %v", err)
	}

	if frame.Length != 5 {
		t.Errorf("expected length 5, got %d", frame.Length)
	}
	if !bytes.Equal(frame.Data, data) {
		t.Errorf("expected data %s, got %s", data, frame.Data)
	}
}
