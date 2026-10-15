package waf

import (
	"bytes"
	"testing"
)

func TestSanitizeStream(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		sensitive [][]byte
		want      string
	}{
		{
			name:      "simple",
			input:     `{"password": "secret123"}`,
			sensitive: [][]byte{[]byte("\"password\"")},
			want:      `{"password": "REDACTED"}`,
		},
		{
			name:      "nested",
			input:     `{"user": {"password": "123"}}`,
			sensitive: [][]byte{[]byte("\"password\"")},
			want:      `{"user": {"password": "REDACTED"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer := NewSanitizer()
			buf := new(bytes.Buffer)
			sanitizer.SanitizeStream(buf, []byte(tt.input), tt.sensitive)
			if buf.String() != tt.want {
				t.Errorf("got %s, want %s", buf.String(), tt.want)
			}
		})
	}
}
