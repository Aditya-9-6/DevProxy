package proxy

import (
	"net/http"
	"time"
)

// NewOptimizedTransport returns an http.Transport configured for high-throughput, low-latency proxying.
func NewOptimizedTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        4096,
		MaxIdleConnsPerHost: 1024,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	}
}
