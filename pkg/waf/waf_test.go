package waf

import (
	"bytes"
	"testing"
)

func TestNormalizeHeaderKey(t *testing.T) {
	tests := []struct {
		input    []byte
		expected []byte
	}{
		{[]byte("content-type"), []byte("Content-Type")},
		{[]byte("x-forwarded-for"), []byte("X-Forwarded-For")},
		{[]byte("host"), []byte("Host")},
		{[]byte("bad\x00key"), []byte("Badkey")},
	}

	for _, tt := range tests {
		got := NormalizeHeaderKey(tt.input)
		if !bytes.Equal(got, tt.expected) {
			t.Errorf("NormalizeHeaderKey(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSanitizeHeaderValue(t *testing.T) {
	tests := []struct {
		input    []byte
		expected []byte
	}{
		{[]byte("value"), []byte("value")},
		{[]byte("value\r\nInjection"), []byte("valueInjection")},
		{[]byte("\n\r"), []byte("")},
	}

	for _, tt := range tests {
		got := SanitizeHeaderValue(tt.input)
		if !bytes.Equal(got, tt.expected) {
			t.Errorf("SanitizeHeaderValue(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func BenchmarkNormalizeHeaderKey(b *testing.B) {
	input := []byte("content-type-header-test")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NormalizeHeaderKey(input)
	}
}
