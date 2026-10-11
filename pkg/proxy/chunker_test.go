package proxy

import (
	"bytes"
	"io"
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
