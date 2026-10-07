package ringbuffer

import (
	"net/http"
	"time"
)

// TrafficEvent represents a single captured HTTP transaction.
type TrafficEvent struct {
	ID             string
	Timestamp      time.Time
	Duration       time.Duration
	ClientIP       string
	Scheme         string
	Host           string
	Path           string
	Method         string
	URL            string
	Proto          string
	StatusCode     int
	ReqHeaders     http.Header
	ReqBody        []byte
	RespHeaders    http.Header
	RespBody       []byte
	TLS            bool
	TLSFingerprint string
}
