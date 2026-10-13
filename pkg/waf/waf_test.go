package waf

import (
	"testing"
)

func TestNormalizeHeaderKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"content-type", "Content-Type"},
		{"x-forwarded-for", "X-Forwarded-For"},
		{"host", "Host"},
		{"bad\x00key", "Badkey"},
	}

	for _, tt := range tests {
		got := NormalizeHeaderKey(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeHeaderKey(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSanitizeHeaderValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"value", "value"},
		{"value\r\nInjection", "valueInjection"},
		{"\n\r", ""},
	}

	for _, tt := range tests {
		got := SanitizeHeaderValue(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeHeaderValue(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}
