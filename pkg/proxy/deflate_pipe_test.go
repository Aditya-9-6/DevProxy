package proxy

import (
	"bytes"
	"compress/flate"
	"io"
	"sync"
	"testing"
)

func TestDeflatePipeCompressionRoundtrip(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	pipe := NewDeflatePipe(flate.DefaultCompression)
	_ = pipe.Close()

	err := pipe.CompressStream(io.Discard, bytes.NewReader([]byte("test")))
	if err != io.ErrClosedPipe {
		t.Errorf("Expected io.ErrClosedPipe, got %v", err)
	}
}

func TestDeflatePipeConcurrentStress(t *testing.T) {
	t.Parallel()
	pipe := NewDeflatePipe(flate.DefaultCompression)
	defer pipe.Close()

	var wg sync.WaitGroup
	workers := 10
	iterations := 20

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := bytes.Repeat([]byte("concurrency stress test payload "), 50)
			for j := 0; j < iterations; j++ {
				var compressed bytes.Buffer
				err := pipe.CompressStream(&compressed, bytes.NewReader(data))
				if err != nil {
					t.Errorf("Concurrent CompressStream error: %v", err)
					return
				}

				var decompressed bytes.Buffer
				err = pipe.DecompressStream(&decompressed, &compressed)
				if err != nil {
					t.Errorf("Concurrent DecompressStream error: %v", err)
					return
				}

				if !bytes.Equal(decompressed.Bytes(), data) {
					t.Errorf("Concurrent roundtrip payload mismatch")
					return
				}
			}
		}()
	}
	wg.Wait()

	bytesProc, chunksProc, errCount := pipe.Stats()
	if bytesProc == 0 || chunksProc == 0 {
		t.Errorf("Expected non-zero stats under concurrent load")
	}
	if errCount != 0 {
		t.Errorf("Expected 0 errors under concurrent load, got %d", errCount)
	}
}

func TestDecompressStreamMalformedInput(t *testing.T) {
	t.Parallel()
	pipe := NewDeflatePipe(flate.DefaultCompression)
	defer pipe.Close()

	malformedData := []byte("this is not a valid compressed flate stream at all...")
	var decompressed bytes.Buffer
	err := pipe.DecompressStream(&decompressed, bytes.NewReader(malformedData))
	// Depending on flate reader implementation, malformed input returns an error or read failure
	if err == nil {
		t.Errorf("Expected error when decompressing malformed input, got nil")
	}
}
