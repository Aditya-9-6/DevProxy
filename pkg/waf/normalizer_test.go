package waf

import "testing"

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"//foo//bar", "/foo/bar"},
		{"\\foo\\bar", "/foo/bar"},
		{"a/b/c", "a/b/c"},
	}

	for _, tt := range tests {
		if got := NormalizePath(tt.input); got != tt.expected {
			t.Errorf("NormalizePath(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDetectTraversal(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"../../etc/passwd", true},
		{"/var/www/html", false},
		{"/foo/../bar", true},
	}

	for _, tt := range tests {
		if got := DetectTraversal(tt.input); got != tt.want {
			t.Errorf("DetectTraversal(%q) = %v; want %v", tt.input, got, tt.want)
		}
	}
}
