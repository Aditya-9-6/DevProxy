package proxy

import (
	"net/http"
	"testing"
)

func TestExtractTraceContext(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		wantParent string
		wantState  string
	}{
		{"Valid", map[string]string{"traceparent": "00-123-456-01", "tracestate": "rojo=00f067aa0ba902b7"}, "00-123-456-01", "rojo=00f067aa0ba902b7"},
		{"Empty", map[string]string{}, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			p, s := extractTraceContext(req)
			if p != tt.wantParent || s != tt.wantState {
				t.Errorf("got %s, %s; want %s, %s", p, s, tt.wantParent, tt.wantState)
			}
		})
	}
}
