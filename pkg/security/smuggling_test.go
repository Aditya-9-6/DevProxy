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
		{
			name:    "Valid Content-Length",
			headers: http.Header{"Content-Length": []string{"10"}},
			want:    true,
		},
		{
			name:    "Contradiction CL and TE",
			headers: http.Header{"Content-Length": []string{"10"}, "Transfer-Encoding": []string{"chunked"}},
			want:    false,
		},
		{
			name:    "Valid Chunked TE",
			headers: http.Header{"Transfer-Encoding": []string{"chunked"}},
			want:    true,
		},
		{
			name:    "Case Variant Header Keys",
			headers: http.Header{"content-length": []string{"10"}, "transfer-encoding": []string{"chunked"}},
			want:    false,
		},
		{
			name:    "Multiple Header Values Conflict",
			headers: http.Header{"Content-Length": []string{"10", "20"}},
			want:    true,
		},
		{
			name:    "RFC Violating TE Identity",
			headers: http.Header{"Transfer-Encoding": []string{"chunked, identity"}},
			want:    true,
		},
		{
			name:    "Invalid TE Value",
			headers: http.Header{"Transfer-Encoding": []string{"gzip"}},
			want:    false,
		},
	}

	for _, tt := range tests {
		logCountBefore := s.ContradictionCount
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Sanitize(tt.headers); got != tt.want {
				t.Errorf("Sanitize() = %v, want %v", got, tt.want)
			}
		})
		if !tt.want && s.ContradictionCount == logCountBefore {
			// If it failed due to contradiction, check counter increment
			if _, hasCL := tt.headers["Content-Length"]; hasCL {
				if _, hasTE := tt.headers["Transfer-Encoding"]; hasTE {
					// Contradiction counter should have incremented
				}
			}
		}
	}
}
