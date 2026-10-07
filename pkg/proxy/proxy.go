package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"net/http"
)

// extractTraceContext pulls W3C headers from the request.
func extractTraceContext(r *http.Request) (string, string) {
	return r.Header.Get("traceparent"), r.Header.Get("tracestate")
}

// Note: In the actual request handling logic (e.g., inside the ServeHTTP or capture logic),
// the event should be populated as follows:
// event.TraceParent, event.TraceState = extractTraceContext(r)
