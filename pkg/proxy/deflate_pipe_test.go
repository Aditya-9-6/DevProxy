package proxy

import (
	"bytes"
	"compress/flate"
	"io"
	"testing"
)

func TestDeflatePipeCompressionRoundtrip(t *testing.T) {
	pipe := NewDeflatePipe(flate.BestSpeed)
	defer pipe.Close()

	originalData := []byte("Hello DevProxy! High-throughput network proxy and zero-allocation streaming engine.")
	inputStr := string(originalData)
	src := bytes.NewReader(originalData)

	var compressed bytes.Buffer
	err := pipe.CompressStream(&compressed, src)
	if err != nil {
		t.Fatalf("CompressStream failed: %v", err)
	}

	var decompressed bytes.Buffer
	err = pipe.DecompressStream(&decompressed, &compressed)
	if err != nil {
		t.Fatalf("DecompressStream failed: %v", err)
	}

	if decompressed.String() != inputStr {
		t.Errorf("Roundtrip mismatch: got %q, want %q", decompressed.String(), inputStr)
	}

	bytesProc, chunksProc, errCount := pipe.Stats()
	if bytesProc == 0 || chunksProc == 0 {
		t.Errorf("Expected non-zero stats: bytes=%d, chunks=%d", bytesProc, chunksProc)
	}
	if errCount != 0 {
		t.Errorf("Expected 0 errors, got %d", errCount)
	}
}

func TestDeflatePipeClosedPipe(t *testing.T) {
	pipe := NewDeflatePipe(flate.DefaultCompression)
	_ = pipe.Close()

	err := pipe.CompressStream(io.Discard, bytes.NewReader([]byte("test")))
	if err != io.ErrClosedPipe {
		t.Errorf("Expected io.ErrClosedPipe, got %v", err)
	}
}
