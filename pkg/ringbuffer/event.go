package ringbuffer

import (
	"net/http"
	"time"
)

// TrafficEvent captures the full request and response pair cloned from the data path.
type TrafficEvent struct {
	ID          string        `json:"id"`
	Timestamp   time.Time     `json:"timestamp"`
	Duration    time.Duration `json:"duration_ns"`
	ClientIP    string        `json:"client_ip"`
	Scheme      string        `json:"scheme"`
	Host        string        `json:"host"`
	Method      string        `json:"method"`
	Path        string        `json:"path"`
	URL         string        `json:"url"`
	Proto       string        `json:"proto"`
	ReqHeaders  http.Header   `json:"req_headers"`
	ReqBody     []byte        `json:"req_body"`
	StatusCode  int           `json:"status_code"`
	RespHeaders http.Header   `json:"resp_headers"`
	RespBody    []byte        `json:"resp_body"`
	TLS         bool          `json:"tls"`
	TLSServer   string        `json:"tls_server,omitempty"`
	TraceParent string        `json:"traceparent,omitempty"`
	TraceState  string        `json:"tracestate,omitempty"`
}
