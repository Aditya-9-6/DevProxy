package waf

import (
	"bytes"
	"testing"
)

func TestNormalizeHeaderKey(t *testing.T) {
	input := []byte("content-type")
	got := NormalizeHeaderKey(input)
	defer PutBuffer(got[:cap(got)])
	expected := []byte("Content-Type")
	if !bytes.Equal(got, expected) {
		t.Errorf("NormalizeHeaderKey(%q) = %q; want %q", input, got, expected)
	}
}

func TestSanitizeHeaderValue(t *testing.T) {
	input := []byte("value\r\nInjection")
	got := SanitizeHeaderValue(input)
	defer PutBuffer(got[:cap(got)])
	expected := []byte("valueInjection")
	if !bytes.Equal(got, expected) {
		t.Errorf("SanitizeHeaderValue(%q) = %q; want %q", input, got, expected)
	}
}

func BenchmarkNormalizeHeaderKey(b *testing.B) {
	input := []byte("content-type-header-test")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		got := NormalizeHeaderKey(input)
		PutBuffer(got[:cap(got)])
	}
}
