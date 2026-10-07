package ringbuffer

import (
	"net/http"
	"time"
)

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
	TLS            bool
	TLSFingerprint string
	DurationMs     float64
}
