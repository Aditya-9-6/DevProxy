package waf

import (
	"testing"
)

func TestNormalizeQueryString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{{"b=2&a=1", "a=1&b=2"}, {"z=1&z=0", "z=0&z=1"}, {"", ""}, {"a=1", "a=1"}}
	for _, tt := range tests {
		got := NormalizeQueryString(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeQueryString(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestEntropyScanner(t *testing.T) {
	s := &EntropyScanner{}
	s.Add([]byte("aaaaa"))
	if s.Calculate() != 0 {
		t.Errorf("Expected 0 entropy for uniform data, got %f", s.Calculate())
	}
}
