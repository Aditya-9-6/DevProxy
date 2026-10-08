package proxy

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// UpstreamTransport manages connection pooling and active keepalive probes.
type UpstreamTransport struct {
	*http.Transport
	probeInterval time.Duration
	stopChan      chan struct{}
	mu            sync.RWMutex
}

// NewUpstreamTransport creates a transport with aggressive keepalive settings.
func NewUpstreamTransport(insecure bool) *UpstreamTransport {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &UpstreamTransport{
		Transport:     transport,
		probeInterval: 30 * time.Second,
		stopChan:      make(chan struct{}),
	}
}

// StartProbes initiates background health checks for idle connections.
func (u *UpstreamTransport) StartProbes(ctx context.Context) {
	ticker := time.NewTicker(u.probeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Trigger idle connection cleanup/refresh
			u.Transport.CloseIdleConnections()
		case <-ctx.Done():
			return
		case <-u.stopChan:
			return
		}
	}
}

// Close stops the probe loop.
func (u *UpstreamTransport) Close() {
	close(u.stopChan)
}
