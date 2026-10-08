package analysis

import (
	"net/http"
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestSmugglingRule_Evaluate(t *testing.T) {
	rule := NewSmugglingRule()

	tests := []struct {
		name           string
		headers        http.Header
		expectedCount  int
		expectedTitles []string
	}{
		{
			name: "Normal Request",
			headers: http.Header{
				"Content-Type":   []string{"application/json"},
				"Content-Length": []string{"123"},
			},
			expectedCount: 0,
		},
		{
			name: "CL.TE Desynchronization",
			headers: http.Header{
				"Content-Length":    []string{"13"},
				"Transfer-Encoding": []string{"chunked"},
			},
			expectedCount:  1,
			expectedTitles: []string{"HTTP Request Smuggling: Content-Length and Transfer-Encoding Both Present"},
		},
		{
			name: "Multiple Conflicting Content-Length Headers",
			headers: http.Header{
				"Content-Length": []string{"10", "20"},
			},
			expectedCount:  1,
			expectedTitles: []string{"Multiple Conflicting Content-Length Headers"},
		},
		{
			name: "Transfer-Encoding Obfuscation",
			headers: http.Header{
				"Transfer-Encoding": []string{"chunked\t"},
			},
			expectedCount:  1,
			expectedTitles: []string{"Transfer-Encoding Header Obfuscation"},
		},
		{
			name: "Header Name Whitespace Obfuscation",
			headers: http.Header{
				"Transfer-Encoding ": []string{"chunked"},
			},
			expectedCount:  1,
			expectedTitles: []string{"HTTP Header Name Whitespace Obfuscation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &ringbuffer.TrafficEvent{
				ID:         "test-req-1",
				Method:     "POST",
				URL:        "https://example.com/api",
				ReqHeaders: tt.headers,
			}

			findings := rule.Evaluate(event)
			if len(findings) != tt.expectedCount {
				t.Fatalf("expected %d findings, got %d", tt.expectedCount, len(findings))
			}

			for i, title := range tt.expectedTitles {
				if findings[i].Title != title {
					t.Errorf("expected finding title %q, got %q", title, findings[i].Title)
				}
			}
		})
	}
}
