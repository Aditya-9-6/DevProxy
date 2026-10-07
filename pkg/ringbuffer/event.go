package ringbuffer

import (
	"net/http"
	"time"
)

// TrafficEvent represents a single captured HTTP transaction.
type TrafficEvent struct {
	ID             string
	Timestamp      time.Time
	Host           string
	Path           string
	Method         string
	URL            string
	ReqHeaders     http.Header
	ReqBody        []byte
	RespHeaders    http.Header
	RespBody       []byte
	StatusCode     int
	TLS            bool
	TLSFingerprint string
	Duration       time.Duration
}
