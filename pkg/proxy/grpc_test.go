package proxy

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestParseGRPCStream(t *testing.T) {
	payload := []byte("hello world")
	buf := new(bytes.Buffer)

	// Write header: compressed=0, length=len(payload)
	buf.WriteByte(0)
	binary.Write(buf, binary.BigEndian, uint32(len(payload)))
	buf.Write(payload)

	frames, err := ParseGRPCStream(buf)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(frames) != 1 {
		t.Errorf("Expected 1 frame, got %d", len(frames))
	}
	if !bytes.Equal(frames[0].Data, payload) {
		t.Errorf("Payload mismatch")
	}
}
