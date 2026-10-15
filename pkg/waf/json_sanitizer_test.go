package waf

import (
	"bytes"
	"strings"
	"testing"
)

func TestSanitizeStream(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		sensitive [][]byte
		want      string
		wantErr   bool
	}{
		{"simple", `{"password": "secret123"}`, [][]byte{[]byte("password")}, `{"password":"REDACTED"}`, false},
		{"nested", `{"user": {"password": "123"}}`, [][]byte{[]byte("password")}, `{"user":{"password":"REDACTED"}}`, false},
		{"malformed", `{"key": "val"`, nil, "", true},
		{"empty", "", nil, "", true},
		{"no_match", `{"a": 1}`, [][]byte{[]byte("b")}, `{"a":1}`, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sanitizer := NewSanitizer()
			buf := new(bytes.Buffer)
			err := sanitizer.SanitizeStream(buf, []byte(tt.input), tt.sensitive)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				got := strings.TrimSpace(buf.String())
				if got != tt.want {
					t.Errorf("got %s, want %s", got, tt.want)
				}
			}
		})
	}
}
