package waf

import "testing"

func TestValidateSQLi(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"SELECT * FROM users", false},
		{"admin' OR 1=1 --", true},
		{"normal_query", false},
	}

	for _, tt := range tests {
		if got := ValidateSQLi(tt.input); got != tt.want {
			t.Errorf("ValidateSQLi(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
