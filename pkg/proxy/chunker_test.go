package proxy

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestChunkingPipeline(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	raw := "ZZZ\r\n"
	decoder := NewChunkDecoder(strings.NewReader(raw))
	_, err := io.ReadAll(decoder)
	if err == nil {
		t.Error("Expected error for malformed header, got nil")
	}
}

func TestMaxChunkSizeLimit(t *testing.T) {
	t.Parallel()
	// Construct a chunk header claiming a size larger than MaxChunkSize (e.g., 2MB)
	raw := "200000\r\n"
	decoder := NewChunkDecoder(strings.NewReader(raw))
	_, err := io.ReadAll(decoder)
	if err == nil {
		t.Error("Expected error when chunk size exceeds MaxChunkSize, got nil")
	}
}

func TestConcurrentChunkDecodingStress(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	concurrency := 50
	iterations := 20

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			raw := "5\r\nHello\r\n5\r\nWorld\r\n0\r\n\r\n"
			for j := 0; j < iterations; j++ {
				decoder := NewChunkDecoder(strings.NewReader(raw))
				output, err := io.ReadAll(decoder)
				if err != nil {
					t.Errorf("Concurrent read failed: %v", err)
					return
				}
				if string(output) != "HelloWorld" {
					t.Errorf("Expected HelloWorld, got %s", string(output))
					return
				}
			}
		}()
	}
	wg.Wait()
}
