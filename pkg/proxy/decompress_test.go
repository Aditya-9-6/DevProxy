package proxy

import (
	"bytes"
	"compress/gzip"
	"io"
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
			if err != nil {
				t.Fatalf("failed to read: %v", err)
			}
			if !bytes.Equal(got, data) {
				t.Errorf("got %s, want %s", got, data)
			}
		})
	}

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
}
