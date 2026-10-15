package waf

import (
	sync "sync"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"//foo//bar", "/foo/bar"},
		{"\\foo\\bar", "/foo/bar"},
		{"a/b/c", "/a/b/c"},
		{"/foo/./bar/../baz", "/foo/baz"},
		{"%2e%2e/etc/passwd", "/etc/passwd"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := NormalizePath(tt.input); got != tt.expected {
				t.Errorf("NormalizePath(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDetectTraversal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"../../etc/passwd", true},
		{"/var/www/html", false},
		{"/foo/../bar", true},
		{"%2e%2e%2fetc%2fpasswd", true},
		{"..%2fetc%2fpasswd", true},
		{"file\x00.txt", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := DetectTraversal(tt.input); got != tt.want {
				t.Errorf("DetectTraversal(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsDoubleEncoded(t *testing.T) {
	t.Parallel()
	if !IsDoubleEncoded("%252e%252e/foo") {
		t.Errorf("Expected double encoded true")
	}
	if IsDoubleEncoded("/foo/bar") {
		t.Errorf("Expected double encoded false")
	}
}

func TestNormalizeQueryString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{{"b=2&a=1", "a=1&b=2"}, {"z=1&z=0", "z=0&z=1"}, {"", ""}, {"a=1", "a=1"}, {"&&a=1&&b=2&", "a=1&b=2"}}
	for _, tt := range tests {
		got := NormalizeQueryString(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeQueryString(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestConcurrentExecution(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	workers := 50
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = NormalizePath("/foo/bar/../baz%2e%2e/test")
				_ = DetectTraversal("%2e%2e/etc/passwd")
				_ = IsDoubleEncoded("%252e%252e")
			}
		}()
	}
	wg.Wait()
}

func TestEntropyScannerConcurrent(t *testing.T) {
	s := &EntropyScanner{}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Add([]byte("testdata"))
			s.Calculate()
		}()
	}
	wg.Wait()
}

func BenchmarkNormalizeQueryString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NormalizeQueryString("c=3&b=2&a=1")
	}
}

func BenchmarkEntropyScanner(b *testing.B) {
	s := &EntropyScanner{}
	data := []byte("benchmark data")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Add(data)
		s.Calculate()
	}
}
