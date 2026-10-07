package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"net/http"
)

// extractTraceContext pulls W3C headers from the request.
func extractTraceContext(r *http.Request) (string, string) {
	return r.Header.Get("traceparent"), r.Header.Get("tracestate")
}

// Within the proxy's request handling logic (e.g., inside the ServeHTTP or capture logic):
// event := &ringbuffer.TrafficEvent{
//     ... existing fields ...
//     TraceParent: r.Header.Get("traceparent"),
//     TraceState:  r.Header.Get("tracestate"),
// }
