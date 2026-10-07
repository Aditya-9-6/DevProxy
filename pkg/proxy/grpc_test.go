package proxy

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestParseGRPCStream(t *testing.T) {
	payload := []byte("hello world")
	buf := new(bytes.Buffer)

	buf.WriteByte(0)
	binary.Write(buf, binary.BigEndian, uint32(len(payload)))
	buf.Write(payload)

	var frames []GRPCFrame
	err := ParseGRPCStream(buf, func(f GRPCFrame) error {
		frames = append(frames, f)
		return nil
	})

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

func TestParseGRPCStream_Limit(t *testing.T) {
	buf := new(bytes.Buffer)
	buf.WriteByte(0)
	binary.Write(buf, binary.BigEndian, uint32(MaxGRPCFrameSize+1))

	err := ParseGRPCStream(buf, func(f GRPCFrame) error { return nil })
	if err == nil {
		t.Error("Expected error for oversized frame, got nil")
	}
}
