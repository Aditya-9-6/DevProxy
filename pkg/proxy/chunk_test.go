package proxy

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestChunkDecoder_Basic(t *testing.T) {
	raw := "5\r\nhello\r\n6\r\n world\r\n0\r\n\r\n"
	decoder := NewChunkDecoder(strings.NewReader(raw))
	out, err := io.ReadAll(decoder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(out))
	}
}

func TestChunkDecoder_SmallReads(t *testing.T) {
	raw := "4\r\nWiki\r\n5\r\npedia\r\n0\r\n\r\n"
	decoder := NewChunkDecoder(strings.NewReader(raw))
	buf := make([]byte, 3)
	var sb strings.Builder
	for {
		n, err := decoder.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if sb.String() != "Wikipedia" {
		t.Fatalf("expected 'Wikipedia', got %q", sb.String())
	}
}

func TestChunkDecoder_InvalidChunkSize(t *testing.T) {
	raw := "XYZ\r\nhello\r\n0\r\n\r\n"
	decoder := NewChunkDecoder(strings.NewReader(raw))
	_, err := io.ReadAll(decoder)
	if err == nil {
		t.Fatalf("expected error for invalid chunk size, got nil")
	}
}

func TestChunkDecoder_ExceedsMaxSize(t *testing.T) {
	raw := "100001\r\n"
	decoder := NewChunkDecoder(strings.NewReader(raw))
	_, err := io.ReadAll(decoder)
	if err == nil {
		t.Fatalf("expected error exceeding max chunk size, got nil")
	}
}

func TestChunkDecoder_ChunkExtensions(t *testing.T) {
	raw := "5;name=value\r\nhello\r\n0\r\n\r\n"
	decoder := NewChunkDecoder(strings.NewReader(raw))
	out, err := io.ReadAll(decoder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "hello" {
		t.Fatalf("expected 'hello', got %q", string(out))
	}
}

func TestChunkDecoder_Close(t *testing.T) {
	raw := "5\r\nhello\r\n5\r\nworld\r\n0\r\n\r\n"
	decoder := NewChunkDecoder(strings.NewReader(raw))
	buf := make([]byte, 2)
	n, err := decoder.Read(buf)
	if err != nil || n != 2 {
		t.Fatalf("read failed: n=%d, err=%v", n, err)
	}
	if err := decoder.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestChunkEncoder_Write(t *testing.T) {
	var buf bytes.Buffer
	enc := NewChunkEncoder(&buf)

	n, err := enc.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Fatalf("write failed: n=%d, err=%v", n, err)
	}
	n, err = enc.Write([]byte(" world"))
	if err != nil || n != 6 {
		t.Fatalf("write failed: n=%d, err=%v", n, err)
	}
	n, err = enc.Write([]byte{})
	if err != nil || n != 0 {
		t.Fatalf("empty write failed: n=%d, err=%v", n, err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	expected := "5\r\nhello\r\n6\r\n world\r\n0\r\n\r\n"
	if buf.String() != expected {
		t.Fatalf("expected %q, got %q", expected, buf.String())
	}
}

func TestSimpleChunkDecoder_Basic(t *testing.T) {
	raw := "5\r\nhello\r\n6\r\n world\r\n0\r\n\r\n"
	decoder := NewSimpleChunkDecoder(strings.NewReader(raw))
	out, err := io.ReadAll(decoder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(out))
	}
}

func TestSimpleChunkDecoder_SmallReads(t *testing.T) {
	raw := "4\r\ntest\r\n0\r\n\r\n"
	decoder := NewSimpleChunkDecoder(strings.NewReader(raw))
	p := make([]byte, 2)
	var sb strings.Builder
	for {
		n, err := decoder.Read(p)
		if n > 0 {
			sb.Write(p[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if sb.String() != "test" {
		t.Fatalf("expected 'test', got %q", sb.String())
	}
}

func TestSimpleChunkDecoder_ConcurrentRead(t *testing.T) {
	raw := "5\r\nhello\r\n5\r\nworld\r\n0\r\n\r\n"
	decoder := NewSimpleChunkDecoder(strings.NewReader(raw))
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 5)
			_, _ = decoder.Read(buf)
		}()
	}
	wg.Wait()
}
