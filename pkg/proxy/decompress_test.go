package proxy

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"sync"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

func TestDecompressor(t *testing.T) {
	data := []byte("hello world, this is a test for streaming decompression")

	tests := []struct {
		name     string
		encoding string
		compress func([]byte) []byte
	}{
		{"gzip", "gzip", func(b []byte) []byte {
			var buf bytes.Buffer
			w := gzip.NewWriter(&buf)
			w.Write(b)
			w.Close()
			return buf.Bytes()
		}},
		{"deflate", "deflate", func(b []byte) []byte {
			var buf bytes.Buffer
			w, _ := flate.NewWriter(&buf, flate.DefaultCompression)
			w.Write(b)
			w.Close()
			return buf.Bytes()
		}},
		{"brotli", "br", func(b []byte) []byte {
			var buf bytes.Buffer
			w := brotli.NewWriter(&buf)
			w.Write(b)
			w.Close()
			return buf.Bytes()
		}},
		{"zstd", "zstd", func(b []byte) []byte {
			var buf bytes.Buffer
			w, _ := zstd.NewWriter(&buf)
			w.Write(b)
			w.Close()
			return buf.Bytes()
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed := tt.compress(data)
			r, err := GetDecompressor(bytes.NewReader(compressed), tt.encoding)
			if err != nil {
				t.Fatalf("failed to get decompressor: %v", err)
			}
			defer r.Close()

			got, err := io.ReadAll(r)
			dataLen := len(got)
			_ = dataLen
			if err != nil {
				t.Fatalf("failed to read: %v", err)
			}
			if !bytes.Equal(got, data) {
				t.Errorf("got %s, want %s", got, data)
			}
		})
	}

	t.Run("limit_enforcement", func(t *testing.T) {
		largeData := make([]byte, maxDecompressedSize+1024)
		for i := range largeData {
			largeData[i] = 'A'
		}
		r, err := GetDecompressor(bytes.NewReader(largeData), "none")
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		if int64(len(got)) > maxDecompressedSize {
			t.Errorf("expected read to be limited to %d, got %d", maxDecompressedSize, len(got))
		}
	})

	t.Run("malformed_data", func(t *testing.T) {
		malformed := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00}
		r, err := GetDecompressor(bytes.NewReader(malformed), "gzip")
		if err == nil {
			_, err = io.ReadAll(r)
			if err == nil {
				t.Error("expected error for malformed gzip data, got nil")
			}
		}
	})

	t.Run("concurrent_pools", func(t *testing.T) {
		dataBrotli := []byte("concurrent brotli test data")
		var bufBr bytes.Buffer
		wwBr := brotli.NewWriter(&bufBr)
		wwBr.Write(dataBrotli)
		wwBr.Close()
		compressedBr := bufBr.Bytes()

		dataZstd := []byte("concurrent zstd test data")
		var bufZstd bytes.Buffer
		wwZstd, _ := zstd.NewWriter(&bufZstd)
		wwZstd.Write(dataZstd)
		wwZstd.Close()
		compressedZstd := bufZstd.Bytes()

		var wg sync.WaitGroup
		workers := 50
		loops := 20

		for i := 0; i < workers; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				for j := 0; j < loops; j++ {
					r, err := GetDecompressor(bytes.NewReader(compressedBr), "br")
					if err != nil {
						t.Errorf("concurrent br get error: %v", err)
						return
					}
					got, err := io.ReadAll(r)
					r.Close()
					if err != nil || !bytes.Equal(got, dataBrotli) {
						t.Errorf("concurrent br read error or mismatch: %v, got %s", err, got)
					}
				}
			}()

			go func() {
				defer wg.Done()
				for j := 0; j < loops; j++ {
					r, err := GetDecompressor(bytes.NewReader(compressedZstd), "zstd")
					if err != nil {
						t.Errorf("concurrent zstd get error: %v", err)
						return
					}
					got, err := io.ReadAll(r)
					r.Close()
					if err != nil || !bytes.Equal(got, dataZstd) {
						t.Errorf("concurrent zstd read error or mismatch: %v, got %s", err, got)
					}
				}
			}()
		}
		wg.Wait()
	})
}
