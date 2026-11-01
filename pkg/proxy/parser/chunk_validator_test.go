package parser

import (
	"bytes"
	"strings"
	"testing"
)

func TestChunkValidator_ValidStream(t *testing.T) {
	raw := "4\r\nWiki\r\n5\r\npedia\r\n0\r\n\r\n"
	val := NewChunkValidator(1024, 5000)
	var buf bytes.Buffer

	n, err := val.ParseStream(strings.NewReader(raw), &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != 9 {
		t.Errorf("expected 9 bytes parsed, got %d", n)
	}

	if buf.String() != "Wikipedia" {
		t.Errorf("expected 'Wikipedia', got %q", buf.String())
	}
}

func TestChunkValidator_MaxChunkSizeExceeded(t *testing.T) {
	raw := "10\r\nTooLargeChunkDataHere\r\n0\r\n\r\n"
	val := NewChunkValidator(2, 100)
	var buf bytes.Buffer

	_, err := val.ParseStream(strings.NewReader(raw), &buf)
	if err == nil {
		t.Fatalf("expected chunk limit exceeded error, got nil")
	}
}

func TestChunkValidator_InvalidHexSize(t *testing.T) {
	raw := "ZZ\r\nData\r\n0\r\n\r\n"
	val := NewChunkValidator(100, 100)
	var buf bytes.Buffer

	_, err := val.ParseStream(strings.NewReader(raw), &buf)
	if err == nil {
		t.Fatalf(
			"expected invalid chunk size error, got nil",
		)
	}
}
