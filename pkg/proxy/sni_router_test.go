package proxy

import (
	"testing"
)

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern string
		domain  string
		want    bool
	}{
		{"*.example.com", "api.example.com", true},
		{"*.example.com", "example.com", false},
		{"example.com", "example.com", true},
	}
	for _, tt := range tests {
		if got := MatchWildcard(tt.pattern, tt.domain); got != tt.want {
			t.Errorf("MatchWildcard(%q, %q) = %v, want %v", tt.pattern, tt.domain, got, tt.want)
		}
	}
}
