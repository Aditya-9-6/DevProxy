package compression

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDictionaryManagerAndCompression(t *testing.T) {
	t.Parallel()
	mgr := GetGlobalManager()
	mgr.RecordLatency(CompressionGzip, 50*time.Microsecond)

	algo := mgr.SelectAlgorithm("application/json", 1024)
	if algo == "" {
		t.Error("expected algorithm selection")
	}

	var buf bytes.Buffer
	wc, err := CompressStream(&buf, CompressionGzip)
	if err != nil {
		t.Fatalf("failed to create compressor: %v", err)
	}
	_, err = wc.Write([]byte("test compression streaming data"))
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := wc.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}
}

func TestCompressionEdgeCasesAndAlgorithms(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		algo    CompressionType
		payload []byte
		hasErr  bool
	}{
		{"GzipNormal", CompressionGzip, []byte("some test payload for gzip compression testing"), false},
		{"BrotliNormal", CompressionBrotli, []byte("some test payload for brotli compression testing"), false},
		{"ZstdNormal", CompressionZstd, []byte("some test payload for zstd compression testing"), false},
		{"IdentityNone", CompressionNone, []byte("some test payload for identity compression testing"), false},
		{"EmptyPayload", CompressionGzip, []byte(""), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			wc, err := CompressStream(&buf, tc.algo)
			if (err != nil) != tc.hasErr {
				t.Fatalf("expected error existence %v, got error %v", tc.hasErr, err)
			}
			if err == nil && wc != nil {
				_, wErr := wc.Write(tc.payload)
				if wErr != nil {
					t.Fatalf("failed writing payload: %v", wErr)
				}
				if cErr := wc.Close(); cErr != nil {
					t.Fatalf("failed closing compressor: %v", cErr)
				}
			}
		})
	}
}

func TestConcurrentStreamingStress(t *testing.T) {
	t.Parallel()
	const workers = 10
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				var buf bytes.Buffer
				wc, err := CompressStream(&buf, CompressionGzip)
				if err != nil {
					t.Errorf("worker %d failed to create stream: %v", id, err)
					return
				}
				_, _ = wc.Write([]byte(strings.Repeat("concurrency stress test payload ", 10)))
				_ = wc.Close()
			}
		}(i)
	}
	wg.Wait()
}
