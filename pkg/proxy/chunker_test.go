package proxy

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestChunkingPipeline(t *testing.T) {
	input := "Hello, DevProxy! This is a chunked stream test."
	var buf bytes.Buffer

	encoder := NewChunkEncoder(&buf)
	_, err := encoder.Write([]byte(input))
	if err != nil {
		t.Fatalf("Encoder failed: %v", err)
	}
	buf.Write([]byte("0\r\n\r\n"))

	decoder := NewChunkDecoder(&buf)
	output, err := io.ReadAll(decoder)
	if err != nil && err != io.EOF {
		t.Fatalf("Decoder failed: %v", err)
	}

	if string(output) != input {
		t.Errorf("Expected %s, got %s", input, string(output))
	}
}

func TestMultiChunk(t *testing.T) {
	raw := "5\r\nHello\r\n5\r\nWorld\r\n0\r\n\r\n"
	decoder := NewChunkDecoder(strings.NewReader(raw))
	output, err := io.ReadAll(decoder)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if string(output) != "HelloWorld" {
		t.Errorf("Expected HelloWorld, got %s", string(output))
	}
}

func TestMalformedHeader(t *testing.T) {
	raw := "ZZZ\r\n"
	decoder := NewChunkDecoder(strings.NewReader(raw))
	_, err := io.ReadAll(decoder)
	if err == nil {
		t.Error("Expected error for malformed header, got nil")
	}
}
