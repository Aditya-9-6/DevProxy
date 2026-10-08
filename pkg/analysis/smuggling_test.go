package analysis

import (
	"net/http"
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestRequestSmugglingRule(t *testing.T) {
	rule := NewRequestSmugglingRule()

	tests := []struct {
		name           string
		headers        http.Header
		expectedCount  int
		expectedTitles []string
	}{
		{
			name: "Normal Request (No Smuggling)",
			headers: http.Header{
				"Content-Length": []string{"15"},
			},
			expectedCount:  0,
			expectedTitles: nil,
		},
		{
			name: "CL.TE Attack Vector (Both Content-Length and Transfer-Encoding: chunked)",
			headers: http.Header{
				"Content-Length":    []string{"13"},
				"Transfer-Encoding": []string{"chunked"},
			},
			expectedCount:  1,
			expectedTitles: []string{"HTTP Request Smuggling: Simultaneous Content-Length and Transfer-Encoding (CL.TE / TE.CL)"},
		},
		{
			name: "Obfuscated Content-Length Header with Space",
			headers: http.Header{
				"Content-Length ": []string{"42"},
			},
			expectedCount:  1,
			expectedTitles: []string{"Obfuscated Content-Length Header"},
		},
		{
			name: "Obfuscated Transfer-Encoding Header with Tab",
			headers: http.Header{
				"Transfer-Encoding\t": []string{"chunked"},
			},
			expectedCount:  1,
			expectedTitles: []string{"Obfuscated Transfer-Encoding Header"},
		},
		{
			name: "Duplicate Content-Length Headers",
			headers: http.Header{
				"Content-Length": []string{"10", "20"},
			},
			expectedCount:  1,
			expectedTitles: []string{"Multiple Content-Length Headers"},
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
			if len(findings) != tc.expectedCount {
				t.Errorf("Expected %d findings, got %d", tc.expectedCount, len(findings))
			}

			for i, title := range tc.expectedTitles {
				if i < len(findings) && findings[i].Title != title {
					t.Errorf("Expected finding title %q, got %q", title, findings[i].Title)
				}
			}
		})
	}
}
