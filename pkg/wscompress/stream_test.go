package wscompress

import (
	"bytes"
	"context"
	"testing"
)

func TestStreamEngineCompressDecompress(t *testing.T) {
	engine := NewStreamEngine(DefaultLimits())
	ctx := context.Background()

	original := []byte("Hello, DevProxy WebSocket permessage-deflate streaming engine! Let's test compression roundtrip.")
	compressed, err := engine.Compress(ctx, bytes.NewReader(original))
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	decompressed, err := engine.Decompress(ctx, compressed)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !bytes.Equal(original, decompressed) {
		t.Fatalf("roundtrip mismatch: got %q, want %q", decompressed, original)
	}
}
