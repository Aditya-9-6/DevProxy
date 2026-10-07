package proxy

import (
	"bytes"
	"testing"
)

func TestParseGRPCFrame(t *testing.T) {
	data := []byte("hello")
	buf := new(bytes.Buffer)
	buf.WriteByte(0)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	buf.Write(lenBuf)
	buf.Write(data)

	frame, err := ParseGRPCFrame(buf)
	if err != nil {
		t.Fatalf("Failed to parse frame: %v", err)
	}

	if frame.Length != uint32(len(data)) {
		t.Errorf("Expected length %d, got %d", len(data), frame.Length)
	}
	if !bytes.Equal(frame.Data, data) {
		t.Errorf("Expected data %s, got %s", data, frame.Data)
	}
}
