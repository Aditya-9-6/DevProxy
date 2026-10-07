package ringbuffer

import (
	"net/http"
	"time"
)

// TrafficEvent represents a single captured HTTP/HTTPS transaction.
type TrafficEvent struct {
	ID             string
	Timestamp      time.Time
	Method         string
	URL            string
	Host           string
	ReqHeaders     http.Header
	ReqBody        []byte
	StatusCode     int
	RespHeaders    http.Header
	RespBody       []byte
	DurationMs     float64
	TLS            bool
	TLSFingerprint string // JA3 hash
}
