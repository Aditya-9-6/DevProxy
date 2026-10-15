package waf

import (
	"bytes"
	"strings"
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
			sensitive: [][]byte{[]byte("password")},
			want:      `{"password":"REDACTED"}`,
		},
		{
			name:      "nested",
			input:     `{"user": {"password": "123"}}`,
			sensitive: [][]byte{[]byte("password")},
			want:      `{"user":{"password":"REDACTED"}}`,
		},
		{
			name:      "escaped",
			input:     `{"key": "val\"ue"}`,
			sensitive: [][]byte{[]byte("key")},
			want:      `{"key":"REDACTED"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer := NewSanitizer()
			buf := new(bytes.Buffer)
			sanitizer.SanitizeStream(buf, []byte(tt.input), tt.sensitive)
			got := strings.TrimSpace(buf.String())
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}
