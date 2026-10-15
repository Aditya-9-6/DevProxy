package security

import (
	"net/http"
	"testing"
)

func TestSmugglingSanitizer(t *testing.T) {
	s := NewSmugglingSanitizer()

	tests := []struct {
		name    string
		headers http.Header
		want    bool
	}{
		{"Valid", http.Header{"Content-Length": []string{"10"}}, true},
		{"Contradiction", http.Header{"Content-Length": []string{"10"}, "Transfer-Encoding": []string{"chunked"}}, false},
		{"ValidChunked", http.Header{"Transfer-Encoding": []string{"chunked"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Sanitize(tt.headers); got != tt.want {
				t.Errorf("Sanitize() = %v, want %v", got, tt.want)
			}
		})
	}
}
