package waf

import (
	"bytes"
	"sync"
	"testing"
)

func TestNormalizeHeaderKey(t *testing.T) {
	input := []byte("content-type")
	pbuf, ok := NormalizeHeaderKey(input)
	if !ok {
		t.Fatal("failed to normalize")
	}
	defer pbuf.Close()
	expected := []byte("Content-Type")
	if !bytes.Equal(pbuf.Data, expected) {
		t.Errorf("got %q; want %q", pbuf.Data, expected)
	}
}

func TestSanitizeHeaderValue(t *testing.T) {
	input := []byte("value\r\nInjection")
	pbuf, ok := SanitizeHeaderValue(input)
	if !ok {
		t.Fatal("failed to sanitize")
	}
	defer pbuf.Close()
	expected := []byte("valueInjection")
	if !bytes.Equal(pbuf.Data, expected) {
		t.Errorf("got %q; want %q", pbuf.Data, expected)
	}
}

func TestConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pbuf, _ := NormalizeHeaderKey([]byte("test-header"))
			pbuf.Close()
		}()
	}
	wg.Wait()
}

func BenchmarkNormalizeHeaderKey(b *testing.B) {
	input := []byte("content-type-header-test")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pbuf, _ := NormalizeHeaderKey(input)
		pbuf.Close()
	}
}
