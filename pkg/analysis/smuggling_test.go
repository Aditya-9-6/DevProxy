package analysis

import (
	"net/http"
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestSmugglingRule(t *testing.T) {
	rule := NewSmugglingRule()

	tests := []struct {
		name             string
		headers          http.Header
		expectedFindings int
		expectedCategory string
	}{
		{
			name: "Normal Request",
			headers: http.Header{
				"Content-Length": []string{"123"},
			},
			expectedFindings: 0,
		},
		{
			name: "CL.TE / TE.CL Conflict",
			headers: http.Header{
				"Content-Length":    []string{"13"},
				"Transfer-Encoding": []string{"chunked"},
			},
			expectedFindings: 1,
			expectedCategory: "REQUEST_SMUGGLING",
		},
		{
			name: "Multiple Content-Length Headers",
			headers: http.Header{
				"Content-Length": []string{"10", "20"},
			},
			expectedFindings: 1,
			expectedCategory: "REQUEST_SMUGGLING",
		},
		{
			name: "Multiple Transfer-Encoding Headers",
			headers: http.Header{
				"Transfer-Encoding": []string{"chunked", "gzip"},
			},
			expectedFindings: 1,
			expectedCategory: "REQUEST_SMUGGLING",
		},
		{
			name: "Obfuscated Transfer-Encoding",
			headers: http.Header{
				"Transfer-Encoding": []string{"xchunked"},
			},
			expectedFindings: 1,
			expectedCategory: "REQUEST_SMUGGLING",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &ringbuffer.TrafficEvent{
				ID:         "test-req-1",
				Method:     "POST",
				URL:        "https://example.com/api",
				ReqHeaders: tc.headers,
			}

			findings := rule.Evaluate(event)
			if len(findings) != tc.expectedFindings {
				t.Errorf("expected %d findings, got %d", tc.expectedFindings, len(findings))
			}

			if tc.expectedFindings > 0 && len(findings) > 0 {
				if findings[0].Category != tc.expectedCategory {
					t.Errorf("expected category %s, got %s", tc.expectedCategory, findings[0].Category)
				}
			}
		})
	}
}
