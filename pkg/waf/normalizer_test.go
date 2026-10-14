package waf

import (
	"sync"
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
