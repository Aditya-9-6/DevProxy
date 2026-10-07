package proxy

import (
	"net/http"
	"testing"
)

func TestExtractTraceContext(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("traceparent", "00-123-456-01")
	req.Header.Set("tracestate", "rojo=00f067aa0ba902b7")

	parent, state := extractTraceContext(req)
	if parent != "00-123-456-01" || state != "rojo=00f067aa0ba902b7" {
		t.Errorf("Failed to extract trace context: %s, %s", parent, state)
	}
}
