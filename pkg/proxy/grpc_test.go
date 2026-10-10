package proxy

import (
	"bytes"
	"testing"
)

func TestGRPCFramingRoundTrip(t *testing.T) {
	originalMsg := &GRPCMessage{
		Compressed: false,
		Data:       []byte("hello grpc stream"),
	}

	var buf bytes.Buffer
	if err := WriteGRPCMessage(&buf, originalMsg); err != nil {
		t.Fatalf("Failed to write gRPC message: %v", err)
	}

	readMsg, err := ReadGRPCMessage(&buf)
	if err != nil {
		t.Fatalf("Failed to read gRPC message: %v", err)
	}

	if readMsg.Compressed != originalMsg.Compressed {
		t.Errorf("Compressed flag mismatch: expected %v, got %v", originalMsg.Compressed, readMsg.Compressed)
	}

	if string(readMsg.Data) != string(originalMsg.Data) {
		t.Errorf("Payload mismatch: expected %s, got %s", string(originalMsg.Data), string(readMsg.Data))
	}
}

<<<<<<< HEAD
func TestGRPCFramingRoundTrip(t *testing.T) {
	originalMsg := &GRPCMessage{
		Compressed: false,
		Data:       []byte("hello grpc stream"),
	}

	var buf bytes.Buffer
	if err := WriteGRPCMessage(&buf, originalMsg); err != nil {
		t.Fatalf("Failed to write gRPC message: %v", err)
	}

	readMsg, err := ReadGRPCMessage(&buf)
	if err != nil {
		t.Fatalf("Failed to read gRPC message: %v", err)
	}

	if readMsg.Compressed != originalMsg.Compressed {
		t.Errorf("Compressed flag mismatch: expected %v, got %v", originalMsg.Compressed, readMsg.Compressed)
	}

	if string(readMsg.Data) != string(originalMsg.Data) {
		t.Errorf("Payload mismatch: expected %s, got %s", string(originalMsg.Data), string(readMsg.Data))
	}
}

func TestGRPCFramingCompressedAndEmpty(t *testing.T) {
	msg := &GRPCMessage{
		Compressed: true,
		Data:       []byte{},
	}

	var buf bytes.Buffer
=======
func TestGRPCFramingCompressedAndEmpty(t *testing.T) {
	msg := &GRPCMessage{
		Compressed: true,
		Data:       []byte{},
	}

	var buf bytes.Buffer
>>>>>>> origin/main
	if err := WriteGRPCMessage(&buf, msg); err != nil {
		t.Fatalf("Failed to write compressed empty message: %v", err)
	}

	readMsg, err := ReadGRPCMessage(&buf)
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if !readMsg.Compressed {
		t.Errorf("Expected compressed=true, got %v", readMsg.Compressed)
	}
	if len(readMsg.Data) != 0 {
		t.Errorf("Expected empty data, got %v", readMsg.Data)
	}
}

func TestGRPCFramingErrors(t *testing.T) {
	if err := WriteGRPCMessage(&bytes.Buffer{}, nil); err == nil {
		t.Error("Expected error writing nil message, got nil")
	}

	// Truncated header
	shortBuf := bytes.NewReader([]byte{0, 0, 0})
	if _, err := ReadGRPCMessage(shortBuf); err == nil {
		t.Errorf("Expected error reading truncated header, got nil")
	}

	// Truncated payload: claims length 10, but only 2 bytes present
	corruptBuf := bytes.NewReader([]byte{0, 0, 0, 0, 10, 1, 2})
	if _, err := ReadGRPCMessage(corruptBuf); err == nil {
		t.Error("Expected error reading truncated payload, got nil")
	}
}
