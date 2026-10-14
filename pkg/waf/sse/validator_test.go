package sse

import (
	"testing"
)

func TestValidator_Validate(t *testing.T) {
	v := NewValidator()
	tests := []struct {
		name  string
		event string
		want  bool
	}{
		{"Valid data", "data: hello\n\n", true},
		{"Valid event and data", "event: update\ndata: {\"id\": 1}\n\n", true},
		{"Forbidden script", "data: <script>alert(1)</script>\n\n", false},
		{"Forbidden javascript", "data: javascript:void(0)\n\n", false},
		{"Forbidden eval", "data: eval(foo)\n\n", false},
		{"Comment line", ": this is a comment\ndata: ok\n\n", true},
		{"Invalid line format", "invalid line here\n\n", false},
		{"Valid retry", "retry: 1000\ndata: ok\n\n", true},
		{"Valid id", "id: 123\ndata: ok\n\n", true},
		{"Empty event", "\n", true},
		{"Multiple colons in data", "data: key:value:pair\n\n", true},
		{"Mixed valid and invalid", "data: ok\nmalformed line\n\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.Validate([]byte(tt.event)); got != tt.want {
				t.Errorf("Validator.Validate(%q) = %v, want %v", tt.event, got, tt.want)
			}
		})
	}
}
