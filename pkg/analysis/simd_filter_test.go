package analysis

import (
	"testing"
)

func TestContainsSecurityMarker(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"clean", "hello world", false},
		{"sqli", "SELECT * FROM users", true},
		{"xss", "<script>alert(1)</script>", true},
		{"partial", "some data OR 1=1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsSecurityMarker([]byte(tt.input)); got != tt.expected {
				t.Errorf("ContainsSecurityMarker() = %v, want %v", got, tt.expected)
			}
		})
	}
}
